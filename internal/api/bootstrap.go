package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func ParseSeedNodes(raw string) ([]Node, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var nodes []Node
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		return nil, fmt.Errorf("parse BOOSTLAB_NODES_JSON: %w", err)
	}

	seen := make(map[string]struct{}, len(nodes))
	for i := range nodes {
		node := &nodes[i]

		node.ID = strings.TrimSpace(node.ID)
		node.Region = strings.TrimSpace(node.Region)
		node.Host = strings.TrimSpace(node.Host)
		node.RouteAPIURL = strings.TrimSpace(node.RouteAPIURL)
		node.WireGuardPublicKey = strings.TrimSpace(node.WireGuardPublicKey)

		if node.ID == "" {
			return nil, fmt.Errorf("node %d: id is required", i)
		}
		if node.Region == "" {
			return nil, fmt.Errorf("node %q: region is required", node.ID)
		}
		if node.Host == "" {
			return nil, fmt.Errorf("node %q: host is required", node.ID)
		}
		if node.UDPPort < 1 || node.UDPPort > 65535 {
			return nil, fmt.Errorf("node %q: invalid udp_port", node.ID)
		}
		if node.RouteAPIURL != "" {
			parsed, err := url.ParseRequestURI(node.RouteAPIURL)
			if err != nil ||
				parsed.Scheme != "https" ||
				parsed.Host == "" {
				return nil, fmt.Errorf("node %q: route_api_url must be a valid HTTPS URL", node.ID)
			}
		}

		if node.WireGuardPublicKey != "" {
			if node.WireGuardPort < 1 || node.WireGuardPort > 65535 {
				return nil, fmt.Errorf("node %q: invalid wireguard_port", node.ID)
			}
			decoded, err := base64.StdEncoding.DecodeString(node.WireGuardPublicKey)
			if err != nil || len(decoded) != 32 {
				return nil, fmt.Errorf("node %q: invalid WireGuard public key", node.ID)
			}
		} else if node.WireGuardPort != 0 {
			return nil, fmt.Errorf("node %q: wireguard_public_key is required when wireguard_port is set", node.ID)
		}

		if _, exists := seen[node.ID]; exists {
			return nil, fmt.Errorf("duplicate node id %q", node.ID)
		}
		seen[node.ID] = struct{}{}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Region == nodes[j].Region {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].Region < nodes[j].Region
	})

	return nodes, nil
}
