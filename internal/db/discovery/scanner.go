package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// Scanner discovers PostgreSQL instances
type Scanner struct {
	timeout time.Duration
}

// NewScanner creates a new scanner
func NewScanner() *Scanner {
	return &Scanner{
		timeout: 2 * time.Second,
	}
}

// ScanPorts scans for PostgreSQL on common ports
func (s *Scanner) ScanPorts(ctx context.Context, host string, ports []int) []models.DiscoveredInstance {
	if len(ports) == 0 {
		ports = []int{5432, 5433, 5434, 5435}
	}

	instances := make([]models.DiscoveredInstance, 0)
	resultChan := make(chan models.DiscoveredInstance, len(ports))

	for _, port := range ports {
		go func(p int) {
			instance := s.scanPort(ctx, host, p)
			resultChan <- instance
		}(port)
	}

	for range ports {
		instance := <-resultChan
		if instance.Available {
			instances = append(instances, instance)
		}
	}

	return instances
}

// scanPort checks if a port is open
func (s *Scanner) scanPort(ctx context.Context, host string, port int) models.DiscoveredInstance {
	instance := models.DiscoveredInstance{
		Host:   host,
		Port:   port,
		Source: models.SourcePortScan,
	}

	start := time.Now()
	address := fmt.Sprintf("%s:%d", host, port)

	dialer := &net.Dialer{
		Timeout: s.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	instance.ResponseTime = time.Since(start)

	if err != nil {
		instance.Available = false
		return instance
	}

	conn.Close()
	instance.Available = true

	return instance
}

// ScanLocalhost scans for PostgreSQL on localhost
func (s *Scanner) ScanLocalhost(ctx context.Context) []models.DiscoveredInstance {
	return s.ScanPorts(ctx, "localhost", []int{5432, 5433, 5434, 5435})
}
