# BOOSTLAB Control

Control-plane service for BOOSTLAB.

## Responsibilities

- Gateway registry and health.
- Client bootstrap configuration.
- Region/server selection metadata.
- Route-quality measurements.
- Future authentication, plans and entitlements.

The control plane never needs to inspect game payloads. Data-plane traffic belongs to the gateway layer.

## Status

Repository initialized. API skeleton and CI follow in the first control-plane stage.
