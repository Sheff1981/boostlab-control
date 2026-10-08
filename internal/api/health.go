package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type GatewayHealthMonitor struct {
	Registry *Registry
	Client   *http.Client
	Log      *slog.Logger
	Interval time.Duration

	mu       sync.Mutex
	failures map[string]int
}

func NewGatewayHealthMonitor(
	registry *Registry,
	log *slog.Logger,
	interval time.Duration,
) *GatewayHealthMonitor {
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}
	return &GatewayHealthMonitor{
		Registry: registry,
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
		Log:      log,
		Interval: interval,
		failures: make(map[string]int),
	}
}

func (m *GatewayHealthMonitor) Run(ctx context.Context) {
	if m == nil || m.Registry == nil {
		return
	}

	m.checkAll(ctx)

	ticker := time.NewTicker(m.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAll(ctx)
		}
	}
}

func (m *GatewayHealthMonitor) checkAll(ctx context.Context) {
	nodes := m.Registry.List()
	for _, node := range nodes {
		url := strings.TrimSpace(node.RouteAPIURL)
		if url == "" {
			continue
		}
		go m.checkOne(ctx, node.ID, strings.TrimRight(url, "/")+"/readyz")
	}
}

func (m *GatewayHealthMonitor) checkOne(ctx context.Context, nodeID, url string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		m.record(nodeID, false)
		return
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := m.Client.Do(req)
	if err != nil {
		m.record(nodeID, false)
		return
	}
	defer resp.Body.Close()

	m.record(nodeID, resp.StatusCode == http.StatusNoContent)
}

func (m *GatewayHealthMonitor) record(nodeID string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		m.failures[nodeID] = 0
		if m.Registry.SetHealthy(nodeID, true) && m.Log != nil {
			m.Log.Debug("gateway health ok", "node_id", nodeID)
		}
		return
	}

	count := m.failures[nodeID] + 1
	m.failures[nodeID] = count
	if count < 2 {
		return
	}

	if m.Registry.SetHealthy(nodeID, false) && m.Log != nil {
		m.Log.Warn("gateway marked unhealthy", "node_id", nodeID, "consecutive_failures", count)
	}
}
