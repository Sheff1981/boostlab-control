package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGatewayHealthMonitorRequiresTwoFailures(t *testing.T) {
	registry := NewRegistry()
	registry.Upsert(Node{
		ID:      "fra-1",
		Region:  "europe",
		Host:    "203.0.113.10",
		UDPPort: 51821,
		Healthy: true,
	})
	monitor := NewGatewayHealthMonitor(registry, nil, 15*time.Second)

	monitor.record("fra-1", false)
	node, _ := registry.Get("fra-1")
	if !node.Healthy {
		t.Fatal("single transient failure should not mark node unhealthy")
	}

	monitor.record("fra-1", false)
	node, _ = registry.Get("fra-1")
	if node.Healthy {
		t.Fatal("two consecutive failures should mark node unhealthy")
	}

	monitor.record("fra-1", true)
	node, _ = registry.Get("fra-1")
	if !node.Healthy {
		t.Fatal("successful readiness check should restore node immediately")
	}
}

func TestGatewayHealthMonitorChecksReadyz(t *testing.T) {
	healthy := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			http.NotFound(w, r)
			return
		}
		if healthy {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Upsert(Node{
		ID:          "fra-1",
		Region:      "europe",
		Host:        "203.0.113.10",
		UDPPort:     51821,
		RouteAPIURL: server.URL,
		Healthy:     true,
	})
	monitor := NewGatewayHealthMonitor(registry, nil, 15*time.Second)

	monitor.checkOne(context.Background(), "fra-1", server.URL+"/readyz")
	node, _ := registry.Get("fra-1")
	if !node.Healthy {
		t.Fatal("expected healthy node")
	}

	healthy = false
	monitor.checkOne(context.Background(), "fra-1", server.URL+"/readyz")
	monitor.checkOne(context.Background(), "fra-1", server.URL+"/readyz")
	node, _ = registry.Get("fra-1")
	if node.Healthy {
		t.Fatal("expected unhealthy node after repeated readiness failure")
	}
}
