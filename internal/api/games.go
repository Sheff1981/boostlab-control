package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type GameCatalogEntry struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

func ParseGameCatalog(raw string) ([]GameCatalogEntry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var games []GameCatalogEntry
	if err := json.Unmarshal([]byte(raw), &games); err != nil {
		return nil, fmt.Errorf("parse BOOSTLAB_GAMES_JSON: %w", err)
	}

	seen := make(map[string]struct{}, len(games))
	for index := range games {
		game := &games[index]
		game.Title = strings.TrimSpace(game.Title)
		if game.Title == "" || len(game.Title) > 120 {
			return nil, fmt.Errorf("game %d: invalid title", index)
		}
		key := strings.ToLower(game.Title)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate game %q", game.Title)
		}
		seen[key] = struct{}{}

		cleanTags := make([]string, 0, len(game.Tags))
		tagSeen := make(map[string]struct{}, len(game.Tags))
		for _, tag := range game.Tags {
			tag = strings.ToUpper(strings.TrimSpace(tag))
			if tag != "HOT" && tag != "NEW" {
				continue
			}
			if _, exists := tagSeen[tag]; exists {
				continue
			}
			tagSeen[tag] = struct{}{}
			cleanTags = append(cleanTags, tag)
		}
		sort.Strings(cleanTags)
		game.Tags = cleanTags
	}

	sort.Slice(games, func(i, j int) bool {
		return strings.ToLower(games[i].Title) < strings.ToLower(games[j].Title)
	})
	return games, nil
}
