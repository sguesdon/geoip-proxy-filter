package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
    // Server configuration
    Server ServerConfig `json:"server"`
    
    // GeoIP configuration
    GeoIP GeoIPConfig `json:"geoip"`
    
    // MaxMind configuration for updates
    MaxMind MaxMindConfig `json:"maxmind"`
    
    // Logging configuration
    Logging LoggingConfig `json:"logging"`

    TmpDir string `json:"tmp_dir"`
}

type ServerConfig struct {
    Host         string        `json:"host"`
    Port         int           `json:"port"`
    ReadTimeout  time.Duration `json:"read_timeout"`
    WriteTimeout time.Duration `json:"write_timeout"`
    IdleTimeout  time.Duration `json:"idle_timeout"`
}

type GeoIPConfig struct {
    DatabasePath    string   `json:"database_path"`
    AllowedCountries []string `json:"allowed_countries"`
    AllowPrivateIPs bool     `json:"allow_private_ips"`
}

type MaxMindConfig struct {
    URL             string `json:"url"`
    AccountID       string `json:"account_id"`
    LicenseKey      string `json:"license_key"`
    DestinationPath string `json:"destination_path"`
    GeolitePrefix   string `json:"geolite_prefix"`
}

type LoggingConfig struct {
    Level       string `json:"level"`
    LogBlocked  bool   `json:"log_blocked"`
    LogAllowed  bool   `json:"log_allowed"`
}

// getEUCountries returns the list of ISO 3166-1 alpha-2 country codes for European Union member states
func getEUCountries() []string {
    return []string{
        "AT", // Austria
        "BE", // Belgium
        "BG", // Bulgaria
        "HR", // Croatia
        "CY", // Cyprus
        "CZ", // Czech Republic
        "DK", // Denmark
        "EE", // Estonia
        "FI", // Finland
        "FR", // France
        "DE", // Germany
        "GR", // Greece
        "HU", // Hungary
        "IE", // Ireland
        "IT", // Italy
        "LV", // Latvia
        "LT", // Lithuania
        "LU", // Luxembourg
        "MT", // Malta
        "NL", // Netherlands
        "PL", // Poland
        "PT", // Portugal
        "RO", // Romania
        "SK", // Slovakia
        "SI", // Slovenia
        "ES", // Spain
        "SE", // Sweden
    }
}

func NewConfig() *Config {
    config := &Config{
        Server: ServerConfig{
            Host:         getEnvString("PROXY_HOST", ""),
            Port:         getEnvInt("PROXY_PORT", 3128),
            ReadTimeout:  getEnvDuration("PROXY_READ_TIMEOUT", 10*time.Second),
            WriteTimeout: getEnvDuration("PROXY_WRITE_TIMEOUT", 10*time.Second),
            IdleTimeout:  getEnvDuration("PROXY_IDLE_TIMEOUT", 60*time.Second),
        },
        GeoIP: GeoIPConfig{
            DatabasePath:     getEnvString("GEOIP_DATABASE_PATH", "/app/GeoLite2-Country.mmdb"),
            AllowedCountries: getEnvStringSlice("GEOIP_ALLOWED_COUNTRIES", getEUCountries()),
            AllowPrivateIPs:  getEnvBool("GEOIP_ALLOW_PRIVATE_IPS", true),
        },
        MaxMind: MaxMindConfig{
            URL:             getEnvString("MAXMIND_URL", "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&suffix=tar.gz&license_key=%s"),
            AccountID:       getEnvString("GEOLITE_ACCOUNT_ID", ""),
            LicenseKey:      getEnvString("GEOLITE_LICENCE_KEY", ""),
            GeolitePrefix:   getEnvString("MAXMIND_GEOLITE_PREFIX", "GeoLite2-Country"),
        },
        Logging: LoggingConfig{
            Level:      getEnvString("LOG_LEVEL", "info"),
            LogBlocked: getEnvBool("LOG_BLOCKED_REQUESTS", true),
            LogAllowed: getEnvBool("LOG_ALLOWED_REQUESTS", false),
        },
        TmpDir: getEnvString("TMP_DIR", "/tmp/"),
    }
    
    return config
}

// Helper functions for environment variables
func getEnvString(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolValue, err := strconv.ParseBool(value); err == nil {
            return boolValue
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
    if value := os.Getenv(key); value != "" {
        return strings.Split(value, ",")
    }
    return defaultValue
}