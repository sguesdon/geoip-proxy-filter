package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sguesdon/geoip-proxy-filter/internal/config"
	"github.com/sguesdon/geoip-proxy-filter/internal/geoip"
	"github.com/sguesdon/geoip-proxy-filter/internal/health"
	"github.com/sguesdon/geoip-proxy-filter/internal/metrics"
)

type loggerAdapter struct{}

func (l *loggerAdapter) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func handleTunneling(w http.ResponseWriter, r *http.Request, reader *geoip.SafeReader, config *config.Config, logger geoip.Logger) {
	start := time.Now()
	metrics.IncrementActiveConnections()
	defer func() {
		metrics.DecrementActiveConnections()
		metrics.RecordProxyDuration("CONNECT", time.Since(start))
	}()

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		http.Error(w, "Cannot resolve host", http.StatusBadRequest)
		return
	}

	if err := geoip.FilterIP(ipAddr.IP, reader, config, logger); err != nil {
		if config.Logging.LogBlocked {
			log.Printf("Blocked CONNECT to %v: %v", ipAddr.IP, err)
		}
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		return
	}

	go func() {
		_, _ = io.Copy(destConn, clientConn)
	}()
	_, _ = io.Copy(clientConn, destConn)
}

func handleHTTPProxy(w http.ResponseWriter, r *http.Request, reader *geoip.SafeReader, config *config.Config, logger geoip.Logger) {
	start := time.Now()
	defer func() {
		metrics.RecordProxyDuration("HTTP", time.Since(start))
	}()

	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	hostOnly, _, err := net.SplitHostPort(host)
	if err != nil {
		hostOnly = host
	}
	ipAddr, err := net.ResolveIPAddr("ip", hostOnly)
	if err != nil {
		metrics.RecordDNSError()
		http.Error(w, "Cannot resolve host", http.StatusBadRequest)
		return
	}

	if err := geoip.FilterIP(ipAddr.IP, reader, config, logger); err != nil {
		if config.Logging.LogBlocked {
			log.Printf("Blocked HTTP request to %v: %v", ipAddr.IP, err)
		}
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	r.RequestURI = "" // obligatoire pour client.Transport
	transport := http.DefaultTransport

	resp, err := transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	config := config.NewConfig()
	logger := &loggerAdapter{}

	reader, err := geoip.NewSafeReader(config.GeoIP.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open MaxMind DB: %v", err)
	}
	defer reader.Close()

	// Health checker
	healthChecker := health.NewHealthChecker(reader)

	// Démarre le watcher de fichier en arrière-plan
	watcher, err := geoip.NewWatcher(reader, logger)
	if err != nil {
		log.Printf("Failed to create file watcher: %v", err)
	} else {
		if err := watcher.Start(); err != nil {
			log.Printf("Failed to start file watcher: %v", err)
		} else {
			defer watcher.Stop()
		}
	}

	// Serveur principal
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	server := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r, reader, config, logger)
			} else {
				handleHTTPProxy(w, r, reader, config, logger)
			}
		}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Serveur de métriques et health checks
	metricsAddr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port+1000)
	metricsServer := &http.Server{
		Addr: metricsAddr,
	}

	// Routes pour le serveur de métriques
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthChecker.Handler())
	mux.HandleFunc("/health/ready", healthChecker.ReadinessHandler())
	mux.HandleFunc("/health/live", healthChecker.LivenessHandler())
	metricsServer.Handler = mux

	// Canal pour gérer l'arrêt gracieux
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Démarrer les serveurs
	go func() {
		log.Printf("Starting proxy server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy server failed: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting metrics server on %s", metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// Attendre le signal d'arrêt
	<-quit
	log.Println("Shutting down servers...")

	// Arrêt gracieux avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Proxy server forced to shutdown: %v", err)
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		log.Printf("Metrics server forced to shutdown: %v", err)
	}

	log.Println("Servers stopped")
}
