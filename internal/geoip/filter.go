package geoip

import (
	"fmt"
	"net"
	"time"

	"github.com/sguesdon/geoip-proxy-filter/internal/config"
	"github.com/sguesdon/geoip-proxy-filter/internal/metrics"
)

var privateBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() {
		return true
	}
	for _, block := range privateBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// FilterIP vérifie si l'IP est autorisée selon la politique GeoIP
func FilterIP(ip net.IP, reader *SafeReader, config *config.Config, logger Logger) error {
	if config.GeoIP.AllowPrivateIPs && IsPrivateIP(ip) {
		metrics.RecordRequest("unknown", "private", "allowed")
		return nil // On laisse passer les IP privées sans filtrage GeoIP
	}

	// Mesurer le temps de lookup GeoIP
	start := time.Now()
	country, err := reader.Country(ip)
	metrics.RecordGeoIPLookup(time.Since(start))

	if err != nil || country == nil {
		metrics.RecordRequest("unknown", "unknown", "blocked")
		return fmt.Errorf("GeoIP lookup failed")
	}

	countryCode := country.Country.IsoCode
	if countryCode == "" {
		countryCode = "unknown"
	}

	// Vérifier si le code pays est dans la liste des pays autorisés
	for _, allowedCountry := range config.GeoIP.AllowedCountries {
		if country.Country.IsoCode == allowedCountry {
			if config.Logging.LogAllowed {
				logger.Printf("Allowed access from %s (country: %s)", ip, country.Country.IsoCode)
			}
			metrics.RecordRequest("unknown", countryCode, "allowed")
			return nil
		}
	}

	metrics.RecordRequest("unknown", countryCode, "blocked")
	return fmt.Errorf("access denied by GeoIP policy for country %s", country.Country.IsoCode)
}