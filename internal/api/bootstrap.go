package api

import (
	"encoding/json"
	"fmt"
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
