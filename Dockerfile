FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache make git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build-prod

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY ./assets/GeoLite2-Country.mmdb /app/GeoLite2-Country.mmdb
COPY --from=builder /app/dist/proxy .
COPY --from=builder /app/dist/updatedb .

EXPOSE 3128

ENTRYPOINT ["./proxy"]
