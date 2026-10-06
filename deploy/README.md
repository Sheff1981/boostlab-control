# Control-plane deployment

The control plane serves the gateway discovery list consumed by the Android client.

## Endpoints

- `GET /healthz`
- `GET /v1/nodes`

## Configuration

Set `BOOSTLAB_NODES_JSON` to a JSON array of gateways.

Example:

    [{"id":"eu-test-1","region":"eu-test","host":"203.0.113.10","udp_port":51821,"healthy":true}]

The Android client only accepts the control API over HTTPS. Put this service behind a TLS reverse proxy or HTTPS load balancer before using automatic discovery outside local development.

Do not expose an unauthenticated write endpoint for gateway registration. For now, node membership is operator-controlled through environment configuration.
