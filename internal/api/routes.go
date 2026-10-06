package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type GameRouteTarget struct {
	ID           string   `json:"id"`
	PackageNames []string `json:"package_names"`
	Host         string   `json:"host"`
	TCPPort      int      `json:"tcp_port"`
}

func ParseGameRouteTargets(raw string) ([]GameRouteTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var targets []GameRouteTarget
	if err := json.Unmarshal([]byte(raw), &targets); err != nil {
		return nil, fmt.Errorf("parse BOOSTLAB_GAME_ROUTES_JSON: %w", err)
	}

	seen := make(map[string]struct{}, len(targets))
	for i := range targets {
		target := &targets[i]
		target.ID = strings.TrimSpace(target.ID)
		target.Host = strings.TrimSpace(target.Host)
		if target.ID == "" || len(target.ID) > 80 {
			return nil, fmt.Errorf("target %d: invalid id", i)
		}
		if target.Host == "" || len(target.Host) > 253 || strings.ContainsAny(target.Host, " /\\") {
			return nil, fmt.Errorf("target %q: invalid host", target.ID)
		}
		if target.TCPPort < 1 || target.TCPPort > 65535 {
			return nil, fmt.Errorf("target %q: invalid tcp_port", target.ID)
		}

		key := strings.ToLower(target.ID)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate target id %q", target.ID)
		}
		seen[key] = struct{}{}

		cleanPackages := make([]string, 0, len(target.PackageNames))
		packageSeen := make(map[string]struct{}, len(target.PackageNames))
		for _, packageName := range target.PackageNames {
			packageName = strings.TrimSpace(packageName)
			if packageName == "" || len(packageName) > 200 || strings.ContainsAny(packageName, " /\\") {
				continue
			}
			if _, exists := packageSeen[packageName]; exists {
				continue
			}
			packageSeen[packageName] = struct{}{}
			cleanPackages = append(cleanPackages, packageName)
		}
		if len(cleanPackages) == 0 {
			return nil, fmt.Errorf("target %q: at least one package_name is required", target.ID)
		}
		sort.Strings(cleanPackages)
		target.PackageNames = cleanPackages
	}

	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	return targets, nil
}

func filterRouteTargets(targets []GameRouteTarget, packageName string) []GameRouteTarget {
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return append([]GameRouteTarget(nil), targets...)
	}
	out := make([]GameRouteTarget, 0)
	for _, target := range targets {
		for _, candidate := range target.PackageNames {
			if candidate == packageName {
				out = append(out, target)
				break
			}
		}
	}
	return out
}
