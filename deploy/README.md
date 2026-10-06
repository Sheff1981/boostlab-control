# Control-plane deployment

The control plane serves the gateway discovery list consumed by the Android client.

## Endpoints

- `GET /healthz`
- `GET /v1/nodes`

## Configuration

Set `BOOSTLAB_NODES_JSON` to a JSON array of gateways.

Probe-only example:

    [{"id":"eu-test-1","region":"eu-test","host":"203.0.113.10","udp_port":51821,"healthy":true}]

After the WireGuard server key exists, publish its **public** key and tunnel port:

    [{"id":"eu-test-1","region":"eu-test","host":"203.0.113.10","udp_port":51821,"wireguard_public_key":"<SERVER_PUBLIC_KEY>","wireguard_port":51820,"healthy":true}]

The Android client automatically carries the WireGuard public key and port forward when that gateway wins route selection.

The Android client only accepts the control API over HTTPS. Put this service behind a TLS reverse proxy or HTTPS load balancer before using automatic discovery outside local development.

Do not expose an unauthenticated write endpoint for gateway registration or client peer allocation. For now, node membership is operator-controlled through environment configuration.

Private WireGuard keys never belong in the control-plane configuration.
