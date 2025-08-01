package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Compteurs de requêtes
    RequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "geoip_proxy_requests_total",
            Help: "Total number of proxy requests",
        },
        []string{"method", "country", "action"}, // action: allowed/blocked
    )

    // Durée des lookups GeoIP
    GeoIPLookupDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "geoip_lookup_duration_seconds",
            Help:    "Time spent on GeoIP lookups",
            Buckets: prometheus.DefBuckets,
        },
    )

    // Statut de la base GeoIP
    GeoIPDatabaseStatus = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "geoip_database_status",
            Help: "Status of GeoIP database (1=healthy, 0=unhealthy)",
        },
    )

    // Timestamp de la dernière mise à jour de la DB
    GeoIPDatabaseLastUpdate = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "geoip_database_last_update_timestamp",
            Help: "Timestamp of last GeoIP database update",
        },
    )

    // Durée des requêtes proxy
    ProxyRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "geoip_proxy_request_duration_seconds",
            Help:    "Time spent processing proxy requests",
            Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
        },
        []string{"method"},
    )

    // Connexions actives
    ActiveConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "geoip_proxy_active_connections",
            Help: "Number of active proxy connections",
        },
    )

    // Erreurs de résolution DNS
    DNSResolutionErrors = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "geoip_proxy_dns_resolution_errors_total",
            Help: "Total number of DNS resolution errors",
        },
    )
)

// RecordGeoIPLookup enregistre une métrique de lookup GeoIP
func RecordGeoIPLookup(duration time.Duration) {
    GeoIPLookupDuration.Observe(duration.Seconds())
}

// RecordRequest enregistre une métrique de requête
func RecordRequest(method, country, action string) {
    RequestsTotal.WithLabelValues(method, country, action).Inc()
}

// RecordProxyDuration enregistre la durée d'une requête proxy
func RecordProxyDuration(method string, duration time.Duration) {
    ProxyRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// UpdateDatabaseStatus met à jour le statut de la base GeoIP
func UpdateDatabaseStatus(healthy bool) {
    if healthy {
        GeoIPDatabaseStatus.Set(1)
        GeoIPDatabaseLastUpdate.SetToCurrentTime()
    } else {
        GeoIPDatabaseStatus.Set(0)
    }
}

// IncrementActiveConnections incrémente le compteur de connexions actives
func IncrementActiveConnections() {
    ActiveConnections.Inc()
}

// DecrementActiveConnections décrémente le compteur de connexions actives
func DecrementActiveConnections() {
    ActiveConnections.Dec()
}

// RecordDNSError enregistre une erreur DNS
func RecordDNSError() {
    DNSResolutionErrors.Inc()
}