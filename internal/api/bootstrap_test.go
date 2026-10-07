package api

import "testing"

func TestParseSeedNodes(t *testing.T) {
	nodes, err := ParseSeedNodes(`[
		{"id":"eu-1","region":"eu-west","host":"203.0.113.10","udp_port":51821,"healthy":true},
		{"id":"eu-2","region":"eu-central","host":"203.0.113.11","udp_port":51821,"healthy":true}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].ID != "eu-2" {
		t.Fatalf("expected deterministic region sort, got %q first", nodes[0].ID)
	}
}

func TestParseSeedNodesRejectsBadPort(t *testing.T) {
	_, err := ParseSeedNodes(`[
		{"id":"bad","region":"test","host":"127.0.0.1","udp_port":0,"healthy":true}
	]`)
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}


func TestParseSeedNodesAcceptsHTTPSRouteAPI(t *testing.T) {
	nodes, err := ParseSeedNodes(`[
		{"id":"eu-1","region":"eu","host":"203.0.113.10","udp_port":51821,"route_api_url":"https://eu-1.example.com","healthy":true}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].RouteAPIURL != "https://eu-1.example.com" {
		t.Fatalf("unexpected route API: %#v", nodes)
	}
}

func TestParseSeedNodesRejectsHTTPRouteAPI(t *testing.T) {
	_, err := ParseSeedNodes(`[
		{"id":"eu-1","region":"eu","host":"203.0.113.10","udp_port":51821,"route_api_url":"http://eu-1.example.com","healthy":true}
	]`)
	if err == nil {
		t.Fatal("expected insecure route API rejection")
	}
}


func TestParseSeedNodesRejectsIncompleteHTTPSRouteAPI(t *testing.T) {
	_, err := ParseSeedNodes(`[
		{"id":"eu-1","region":"eu","host":"203.0.113.10","udp_port":51821,"route_api_url":"https://","healthy":true}
	]`)
	if err == nil {
		t.Fatal("expected malformed HTTPS route API rejection")
	}
}


func TestParseSeedNodesPreservesLocationMetadata(t *testing.T) {
	nodes, err := ParseSeedNodes(`[
		{"id":"fra-1","region":"europe","country_code":"de","city":"Frankfurt","display_name":"DE - Frankfurt","host":"203.0.113.10","udp_port":51821,"healthy":true}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 ||
		nodes[0].CountryCode != "DE" ||
		nodes[0].City != "Frankfurt" ||
		nodes[0].DisplayName != "DE - Frankfurt" {
		t.Fatalf("unexpected location metadata: %#v", nodes)
	}
}
