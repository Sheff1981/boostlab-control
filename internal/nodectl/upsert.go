package nodectl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sheff1981/boostlab-control/internal/api"
)

func Upsert(existingRaw string, node api.Node) (string, error) {
	existing, err := api.ParseSeedNodes(existingRaw)
	if err != nil {
		return "", err
	}

	node.ID = strings.TrimSpace(node.ID)
	if node.ID == "" {
		return "", fmt.Errorf("node id is required")
	}

	replaced := false
	for index := range existing {
		if existing[index].ID == node.ID {
			existing[index] = node
			replaced = true
			break
		}
	}
	if !replaced {
		existing = append(existing, node)
	}

	raw, err := json.Marshal(existing)
	if err != nil {
		return "", fmt.Errorf("encode nodes: %w", err)
	}

	normalized, err := api.ParseSeedNodes(string(raw))
	if err != nil {
		return "", err
	}
	output, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode normalized nodes: %w", err)
	}
	return string(output), nil
}
