package api

import "testing"

func TestParseGameRouteTargets(t *testing.T) {
	targets, err := ParseGameRouteTargets(`[
		{"id":"pubg-eu","package_names":["com.tencent.ig","com.pubg.imobile"],"host":"example.com","tcp_port":443}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || len(targets[0].PackageNames) != 2 {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestFilterRouteTargetsByPackage(t *testing.T) {
	targets, err := ParseGameRouteTargets(`[
		{"id":"a","package_names":["game.a"],"host":"a.example.com","tcp_port":443},
		{"id":"b","package_names":["game.b"],"host":"b.example.com","tcp_port":443}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	filtered := filterRouteTargets(targets, "game.b")
	if len(filtered) != 1 || filtered[0].ID != "b" {
		t.Fatalf("unexpected filtered targets: %#v", filtered)
	}
}
