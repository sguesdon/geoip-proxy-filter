package health

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/sguesdon/geoip-proxy-filter/internal/geoip"
	"github.com/sguesdon/geoip-proxy-filter/internal/metrics"
)

type HealthStatus struct {
    Status    string            `json:"status"`
    Timestamp time.Time         `json:"timestamp"`
    Checks    map[string]Check  `json:"checks"`
}

type Check struct {
    Status  string        `json:"status"`
    Message string        `json:"message,omitempty"`
    Latency time.Duration `json:"latency_ms"`
}

type HealthChecker struct {
    reader *geoip.SafeReader
}

func NewHealthChecker(reader *geoip.SafeReader) *HealthChecker {
    return &HealthChecker{
        reader: reader,
    }
}

// Handler retourne un handler HTTP pour les health checks
func (h *HealthChecker) Handler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        status := h.GetHealthStatus()
        
        w.Header().Set("Content-Type", "application/json")
        
        // Définir le code de statut HTTP
        if status.Status == "healthy" {
            w.WriteHeader(http.StatusOK)
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        
        json.NewEncoder(w).Encode(status)
    }
}

// ReadinessHandler pour Kubernetes readiness probe
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Test simple de la base GeoIP
        _, err := h.reader.Country(net.ParseIP("8.8.8.8"))
        latency := time.Since(start)
        
        if err != nil {
            metrics.UpdateDatabaseStatus(false)
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("GeoIP database unavailable"))
            return
        }
        
        metrics.UpdateDatabaseStatus(true)
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
        
        // Log de la latency si elle est élevée
        if latency > 100*time.Millisecond {
            // Ici tu peux logger si nécessaire
        }
    }
}

// LivenessHandler pour Kubernetes liveness probe
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Check basique que l'application répond
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }
}

// GetHealthStatus effectue tous les checks de santé
func (h *HealthChecker) GetHealthStatus() HealthStatus {
    status := HealthStatus{
        Timestamp: time.Now(),
        Checks:    make(map[string]Check),
    }
    
    overallHealthy := true
    
    // Check GeoIP Database
    geoipCheck := h.checkGeoIPDatabase()
    status.Checks["geoip_database"] = geoipCheck
    if geoipCheck.Status != "healthy" {
        overallHealthy = false
    }
    
    // Check DNS Resolution
    dnsCheck := h.checkDNSResolution()
    status.Checks["dns_resolution"] = dnsCheck
    if dnsCheck.Status != "healthy" {
        overallHealthy = false
    }
    
    if overallHealthy {
        status.Status = "healthy"
    } else {
        status.Status = "unhealthy"
    }
    
    return status
}

func (h *HealthChecker) checkGeoIPDatabase() Check {
    start := time.Now()
    
    // Test avec plusieurs IPs connues
    testIPs := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}
    
    for _, ipStr := range testIPs {
        ip := net.ParseIP(ipStr)
        if ip == nil {
            continue
        }
        
        country, err := h.reader.Country(ip)
        if err != nil {
            metrics.UpdateDatabaseStatus(false)
            return Check{
                Status:  "unhealthy",
                Message: "GeoIP lookup failed: " + err.Error(),
                Latency: time.Since(start),
            }
        }
        
        if country == nil || country.Country.IsoCode == "" {
            metrics.UpdateDatabaseStatus(false)
            return Check{
                Status:  "unhealthy",
                Message: "GeoIP returned empty country data",
                Latency: time.Since(start),
            }
        }
    }
    
    metrics.UpdateDatabaseStatus(true)
    return Check{
        Status:  "healthy",
        Message: "GeoIP database operational",
        Latency: time.Since(start),
    }
}

func (h *HealthChecker) checkDNSResolution() Check {
    start := time.Now()
    
    // Test de résolution DNS
    testHosts := []string{"google.com", "cloudflare.com"}
    
    for _, host := range testHosts {
        _, err := net.ResolveIPAddr("ip", host)
        if err != nil {
            return Check{
                Status:  "unhealthy",
                Message: "DNS resolution failed for " + host + ": " + err.Error(),
                Latency: time.Since(start),
            }
        }
    }
    
    return Check{
        Status:  "healthy",
        Message: "DNS resolution operational",
        Latency: time.Since(start),
    }
}