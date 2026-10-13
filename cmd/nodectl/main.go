package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Sheff1981/boostlab-control/internal/api"
	"github.com/Sheff1981/boostlab-control/internal/nodectl"
)

func main() {
	var node api.Node
	var existing string

	flag.StringVar(&existing, "existing", os.Getenv("BOOSTLAB_NODES_JSON"), "existing BOOSTLAB_NODES_JSON")
	flag.StringVar(&node.ID, "id", "", "node id, e.g. fra-1")
	flag.StringVar(&node.Region, "region", "", "region, e.g. europe")
	flag.StringVar(&node.CountryCode, "country", "", "ISO country code, e.g. DE")
	flag.StringVar(&node.City, "city", "", "city, e.g. Frankfurt")
	flag.StringVar(&node.DisplayName, "name", "", "display name")
	flag.StringVar(&node.Host, "host", "", "public host or IP")
	flag.IntVar(&node.UDPPort, "udp-port", 51821, "UDP probe port")
	flag.StringVar(&node.RouteAPIURL, "route-api", "", "HTTPS Gateway API URL")
	flag.StringVar(&node.WireGuardPublicKey, "wg-key", "", "WireGuard public key")
	flag.IntVar(&node.WireGuardPort, "wg-port", 51820, "WireGuard UDP port")
	flag.Parse()

	node.Healthy = true

	result, err := nodectl.Upsert(existing, node)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nodectl:", err)
		os.Exit(2)
	}

	fmt.Println("BOOSTLAB_NODES_JSON=" + result)
}
