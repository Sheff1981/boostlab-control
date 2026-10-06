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


## Monetization policy

The control plane now exposes `GET /v1/client-policy`.

The current default is a Free tier with advertising enabled and a Premium tier with advertising disabled. During technical testing, Free is not intentionally given a worse route-quality algorithm; production capacity/priority rules will be added only after real infrastructure measurements exist.

This endpoint is configuration, not payment authentication. Production Premium entitlement will require authenticated, server-authoritative verification.
