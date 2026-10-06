package api

import "testing"

func TestParseGameCatalog(t *testing.T) {
	games, err := ParseGameCatalog(`[
		{"title":"Game B","tags":["new","unknown"]},
		{"title":"Game A","tags":["HOT","hot"]}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 || games[0].Title != "Game A" || games[1].Title != "Game B" {
		t.Fatalf("unexpected games: %#v", games)
	}
	if len(games[0].Tags) != 1 || games[0].Tags[0] != "HOT" {
		t.Fatalf("unexpected tags: %#v", games[0].Tags)
	}
}

func TestParseGameCatalogRejectsDuplicate(t *testing.T) {
	_, err := ParseGameCatalog(`[{"title":"Game"},{"title":"game"}]`)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
