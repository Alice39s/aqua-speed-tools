package utils

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

var (
	defaultResolver *DNSResolver
)

// DNSResolver represents a DNS resolver using DNS over HTTPS
type DNSResolver struct {
	endpoint string
	timeout  time.Duration
	retries  int
	client   *dns.Client
}

// NewDNSResolver creates a new DNS resolver
func NewDNSResolver(endpoint string, timeoutSeconds int, retries int) (*DNSResolver, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	return &DNSResolver{
		endpoint: endpoint,
		timeout:  time.Duration(timeoutSeconds) * time.Second,
		retries:  retries,
		client:   new(dns.Client),
	}, nil
}

// SetDNSResolver sets the default DNS resolver
func SetDNSResolver(resolver *DNSResolver) {
	defaultResolver = resolver
}

// GetDNSResolver returns the default DNS resolver
func GetDNSResolver() *DNSResolver {
	return defaultResolver
}

// Resolve resolves a hostname to its IP addresses
func (r *DNSResolver) Resolve(hostname string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	var ips []net.IP
	var lastErr error

	for attempt := 0; attempt <= r.retries; attempt++ {
		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn(hostname), dns.TypeA)

		resp, _, err := r.client.ExchangeContext(ctx, msg, r.endpoint)
		if err != nil {
			lastErr = err
			if attempt < r.retries {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			break
		}

		for _, ans := range resp.Answer {
			if a, ok := ans.(*dns.A); ok {
				ips = append(ips, a.A)
			}
		}

		if len(ips) > 0 {
			return ips, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("DNS resolution failed after %d attempts: %v", r.retries+1, lastErr)
	}

	return nil, fmt.Errorf("no A records found for %s", hostname)
}
