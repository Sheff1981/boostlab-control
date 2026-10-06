#!/usr/bin/env sh
set -eu

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required" >&2
  exit 1
fi

SECRET="$(openssl rand -hex 32)"
printf '%s
' "BOOSTLAB_TURN_SECRET=$SECRET"
printf '%s
' "coturn: static-auth-secret=$SECRET"
