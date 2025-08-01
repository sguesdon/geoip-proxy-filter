.PHONY: build clean test run

build:
	go build -o dist/proxy ./cmd/proxy/
	go build -o dist/updatedb ./cmd/updatedb/

build-prod:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-a -installsuffix cgo \
		-trimpath \
		-o dist/proxy ./cmd/proxy/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-a -installsuffix cgo \
		-trimpath \
		-o dist/updatedb ./cmd/updatedb/
		
test:
	go test ./...

clean:
	rm -f dist/proxy dist/updatedb

run: build
	./dist/proxy

update-db: build
	./dist/updatedb

dev:
	go run ./cmd/proxy/

.PHONY: docker-build docker-run

docker-build:
	docker build -t geoip-proxy-filter .

docker-run:
	docker run -p 3128:3128 -p 4128:4128 geoip-proxy-filter