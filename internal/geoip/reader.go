package geoip

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type SafeReader struct {
	mu     sync.RWMutex
	reader *geoip2.Reader
	path   string
}

func NewSafeReader(path string) (*SafeReader, error) {
	reader, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoIP database: %w", err)
	}

	return &SafeReader{
		reader: reader,
		path:   path,
	}, nil
}

func (s *SafeReader) Country(ip net.IP) (*geoip2.Country, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reader.Country(ip)
}

func (s *SafeReader) Reload() error {
	newReader, err := geoip2.Open(s.path)
	if err != nil {
		return fmt.Errorf("failed to reload GeoIP database: %w", err)
	}

	s.mu.Lock()
	oldReader := s.reader
	s.reader = newReader
	s.mu.Unlock()

	// Fermer l'ancien reader après un délai pour éviter les races
	go func() {
		time.Sleep(5 * time.Second)
		oldReader.Close()
	}()

	return nil
}

func (s *SafeReader) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reader.Close()
}

func (s *SafeReader) Path() string {
	return s.path
}
