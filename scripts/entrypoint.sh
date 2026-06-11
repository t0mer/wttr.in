#!/bin/sh
set -eu

MMDB_PATH="/app/data/GeoLite2-City.mmdb"
MMDB_URL="https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

if [ ! -f "$MMDB_PATH" ]; then
    echo "Downloading GeoLite2-City database..."
    mkdir -p "$(dirname "$MMDB_PATH")"
    curl -fsSL "$MMDB_URL" -o "$MMDB_PATH"
    echo "GeoLite2-City database saved to $MMDB_PATH"
else
    echo "GeoLite2-City database already present at $MMDB_PATH"
fi

exec "$@"
