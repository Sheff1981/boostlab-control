# Control-plane deployment

BOOSTLAB Control serves gateway discovery, route targets, device authentication and short-lived peer provisioning tickets.

## Main endpoints

- `GET /healthz`
- `GET /v1/nodes`
- `GET /v1/games`
- `GET /v1/route-targets`
- `POST /v1/auth/challenge`
- `POST /v1/auth/session`
- `POST /v1/provision/{node}`

Chat/voice endpoints are separate and do not participate in VPN consensus or route selection.

## HTTPS

Android accepts Control only over HTTPS. Run the service behind a TLS reverse proxy or HTTPS load balancer.

## Device enrollment

A new Android phone generates a non-exportable ECDSA P-256 authentication key in Android Keystore.

For the first authentication only, the user enters `BOOSTLAB_ENROLLMENT_CODE`. After a valid signed challenge, Control stores the device public key in:

    /var/lib/boostlab-control/devices.json

The phone no longer needs the enrollment code after that. Generate a strong code, for example:

    openssl rand -hex 12

If the enrollment code is empty, already enrolled devices can still authenticate but new devices cannot enroll.

## Gateway provisioning

Generate a server-to-server secret once:

    openssl rand -hex 32

Set the same value as `BOOSTLAB_PROVISIONING_SECRET` on Control and Gateway.

After device authentication, Control issues a 90-second HMAC ticket bound to:

- the authenticated device ID;
- one Gateway node ID;
- one WireGuard client public key.

The shared provisioning secret is never sent to Android.

## Node configuration

`BOOSTLAB_NODES_JSON` remains operator-controlled. A production node entry should publish only public data:

    [{"id":"fra-1","region":"europe","country_code":"DE","city":"Frankfurt","display_name":"DE - Frankfurt","host":"203.0.113.10","udp_port":51821,"route_api_url":"https://fra-1.example.com","wireguard_public_key":"<SERVER_PUBLIC_KEY>","wireguard_port":51820,"healthy":true}]

Never place WireGuard private keys or the provisioning secret in this JSON.

## Persistent state

The systemd unit creates `/var/lib/boostlab-control` for writable state while retaining `ProtectSystem=strict`.

Recommended paths:

- `BOOSTLAB_SOCIAL_DATA_FILE=/var/lib/boostlab-control/social.json`
- `BOOSTLAB_DEVICE_AUTH_DATA_FILE=/var/lib/boostlab-control/devices.json`
