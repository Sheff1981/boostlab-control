package nodectl

import (
	"strings"
	"testing"

	"github.com/Sheff1981/boostlab-control/internal/api"
)

func TestUpsertAddsNodeAndKeepsValidJson(t *testing.T) {
	existing := `[{"id":"fra-1","region":"europe","country_code":"DE","city":"Frankfurt","host":"203.0.113.10","udp_port":51821,"healthy":true}]`

	result, err := Upsert(existing, api.Node{
		ID:          "lax-1",
		Region:      "north-america",
		CountryCode: "US",
		City:        "Los Angeles",
		DisplayName: "US - Los Angeles",
		Host:        "198.51.100.20",
		UDPPort:     51821,
		Healthy:     true,
	})
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := api.ParseSeedNodes(result)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if !strings.Contains(result, "Los Angeles") {
		t.Fatalf("expected new city in output: %s", result)
	}
}

func TestUpsertReplacesExistingNodeById(t *testing.T) {
	existing := `[{"id":"fra-1","region":"europe","country_code":"DE","city":"Frankfurt","host":"203.0.113.10","udp_port":51821,"healthy":true}]`

	result, err := Upsert(existing, api.Node{
		ID:          "fra-1",
		Region:      "europe",
		CountryCode: "DE",
		City:        "Frankfurt",
		DisplayName: "DE - Frankfurt 2",
		Host:        "203.0.113.99",
		UDPPort:     51821,
		Healthy:     true,
	})
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := api.ParseSeedNodes(result)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected replacement, got %d nodes", len(nodes))
	}
	if nodes[0].Host != "203.0.113.99" {
		t.Fatalf("unexpected host: %s", nodes[0].Host)
	}
}

func TestUpsertRejectsInvalidNode(t *testing.T) {
	_, err := Upsert("", api.Node{
		ID:      "bad",
		Region:  "europe",
		Host:    "",
		UDPPort: 51821,
		Healthy: true,
	})
	if err == nil {
		t.Fatal("expected invalid node to be rejected")
	}
}
