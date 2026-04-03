package localproxy

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"flowpilot/internal/models"
)

func TestEndpointReturnsAuthenticatedLocalProxyConfig(t *testing.T) {
	m := NewManager(time.Minute)
	defer m.Stop()

	cfg := models.ProxyConfig{Server: "proxy.example:8080", Protocol: models.ProxyHTTP}
	local, err := m.Endpoint(cfg)
	if err != nil {
		t.Fatalf("Endpoint: %v", err)
	}
	if !strings.HasPrefix(local.Server, "127.0.0.1:") {
		t.Fatalf("expected localhost endpoint, got %s", local.Server)
	}
	if local.Protocol != models.ProxySOCKS5 {
		t.Fatalf("expected socks5 local endpoint, got %s", local.Protocol)
	}
	if local.Username == "" || local.Password == "" {
		t.Fatal("expected generated local credentials")
	}
}

func TestEndpointReusesCredentialsWhileEndpointAlive(t *testing.T) {
	m := NewManager(time.Minute)
	defer m.Stop()

	cfg := models.ProxyConfig{Server: "proxy.example:8080", Protocol: models.ProxyHTTP}
	first, err := m.Endpoint(cfg)
	if err != nil {
		t.Fatalf("first Endpoint: %v", err)
	}
	second, err := m.Endpoint(cfg)
	if err != nil {
		t.Fatalf("second Endpoint: %v", err)
	}
	if first.Server != second.Server {
		t.Fatalf("expected same local endpoint, got %s and %s", first.Server, second.Server)
	}
	if first.Username != second.Username || first.Password != second.Password {
		t.Fatal("expected local credentials to be reused for cached endpoint")
	}
}

func TestManagerStatsAfterEndpoint(t *testing.T) {
	m := NewManager(time.Minute)
	defer m.Stop()

	cfg := models.ProxyConfig{Server: "proxy.example:8080", Protocol: models.ProxyHTTP}
	_, err := m.Endpoint(cfg)
	if err != nil {
		t.Fatalf("Endpoint: %v", err)
	}
	_, _ = m.Endpoint(cfg) // reuse
	stats := m.Stats()
	if stats.ActiveEndpoints != 1 {
		t.Errorf("ActiveEndpoints: got %d, want 1", stats.ActiveEndpoints)
	}
	if stats.EndpointCreations != 1 {
		t.Errorf("EndpointCreations: got %d, want 1", stats.EndpointCreations)
	}
	if stats.EndpointReuses != 1 {
		t.Errorf("EndpointReuses: got %d, want 1", stats.EndpointReuses)
	}
}

func TestManagerRecordFailures(t *testing.T) {
	m := NewManager(time.Minute)
	defer m.Stop()

	m.RecordAuthFailure(fmt.Errorf("auth err"))
	m.RecordUpstreamFailure(fmt.Errorf("upstream err"))
	stats := m.Stats()
	if stats.AuthFailures != 1 {
		t.Errorf("AuthFailures: got %d, want 1", stats.AuthFailures)
	}
	if stats.UpstreamFailures != 1 {
		t.Errorf("UpstreamFailures: got %d, want 1", stats.UpstreamFailures)
	}
}

func TestManagerEndpointAddrAfterCreate(t *testing.T) {
	m := NewManager(time.Minute)
	defer m.Stop()

	cfg := models.ProxyConfig{Server: "proxy.example:8080", Protocol: models.ProxyHTTP}
	local, _ := m.Endpoint(cfg)
	_ = local

	addr := m.EndpointAddr(cfg)
	if addr == "" {
		t.Fatal("expected non-empty addr for registered endpoint")
	}
	unknown := models.ProxyConfig{Server: "unknown.host:9999", Protocol: models.ProxyHTTP}
	if m.EndpointAddr(unknown) != "" {
		t.Error("expected empty addr for unknown endpoint")
	}
}

func TestManagerEndpointEmptyServer(t *testing.T) {
	m := NewManager(time.Minute)
	defer m.Stop()

	cfg := models.ProxyConfig{Server: "", Protocol: models.ProxyHTTP}
	result, err := m.Endpoint(cfg)
	if err != nil {
		t.Fatalf("Endpoint with empty server: %v", err)
	}
	if result.Server != "" {
		t.Errorf("expected passthrough (empty server), got %q", result.Server)
	}
}

func TestManagerStopIdempotent(t *testing.T) {
	m := NewManager(time.Minute)
	m.Stop()
	m.Stop() // should not panic
}
