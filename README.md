# 🚧 Work in Progress 🚧

This project is currently under active development.  
Features, structure, and documentation are likely to change frequently.  
Use at your own risk, and feel free to follow along as it evolves!

# geoip-proxy-filter

A lightweight HTTP(S) proxy that filters outgoing traffic based on destination IP geolocation (GeoIP).

## Features

* Supports HTTP and HTTPS via CONNECT tunneling
* GeoIP-based filtering using MaxMind databases
* Allows traffic only to allowed countries (e.g., FR, DE)
* Rejects or logs connections to other geolocations
* Automatically allows private IPs (RFC1918, localhost, etc.)
* Simple and performant Go implementation
* Prometheus metrics for monitoring (exposed on port 4128)
* Automatic GeoIP database updates via dedicated tool
* Docker support with multi-stage builds

## Prerequisites

* Go 1.24.4 or later
* MaxMind GeoLite2 Country database (included in `assets/`)

## Quick Start

### Using Make (recommended)

```bash
# Build the proxy
make build

# Run the proxy
make run

# Or run in development mode
make dev
```

### Manual build and run

```bash
# Build
go build -o dist/proxy ./cmd/proxy/

# Run (GeoIP database is included in assets/)
./dist/proxy
```

By default, the proxy listens on port `3128` and metrics are exposed on port `4128`.

## Example Usage

### Test the proxy with curl

```bash
# Test with a French website (should work)
curl -x http://localhost:3128 https://orange.fr

# Test with a blocked country (will be rejected)
curl -x http://localhost:3128 https://google.com
```

### Monitor metrics

```bash
# View Prometheus metrics
curl http://localhost:4128/metrics
```

## Configuration

The proxy can be configured through environment variables or command-line flags:

* **Allowed countries**: Currently hardcoded to FR and DE in the source code
* **Proxy port**: Default `:3128` 
* **Metrics port**: Default `:4128`
* **GeoIP database**: Uses the included `assets/GeoLite2-Country.mmdb`

## Database Updates

Update the GeoIP database using the included tool:

```bash
# Build the update tool
make build

# Update the database
make update-db
# or
./dist/updatedb
```

## Building

### Development build
```bash
make build
```

### Production build (optimized for Linux containers)
```bash
make build-prod
```

### Docker build
```bash
make docker-build
make docker-run
```

## Deployment

### Docker

The project includes a multi-stage Dockerfile for production deployment:

```bash
# Build the Docker image
docker build -t geoip-proxy-filter .

# Run the container
docker run -p 3128:3128 -p 4128:4128 geoip-proxy-filter
```

### Kubernetes

Work in progress

## Development

### Project Structure

```
├── cmd/
│   ├── proxy/          # Main proxy application
│   └── updatedb/       # GeoIP database updater
├── internal/
│   ├── config/         # Configuration management
│   ├── geoip/          # GeoIP filtering logic
│   ├── health/         # Health checks
│   └── metrics/        # Prometheus metrics
├── assets/             # GeoIP database
└── dist/              # Build outputs
```

### Testing

```bash
make test
```

## License

MIT License - see [LICENCE](LICENCE) file for details.

## Credits

* [MaxMind](https://www.maxmind.com/) for the GeoIP database
* Built with Go and Prometheus for reliable proxy filtering

---

Feel free to open issues or contribute improvements!
