#!/bin/sh
set -eu

MMDB_PATH="/app/data/GeoLite2-City.mmdb"
MMDB_DIR="/app/data"

download_geolite2() {
    echo "Downloading GeoLite2-City database..."
    mkdir -p "$MMDB_DIR"

    tmpdir="$(mktemp -d)"
    # shellcheck disable=SC2064
    trap "rm -rf '$tmpdir'" EXIT

    curl -fsSL \
        "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz" \
        -o "$tmpdir/GeoLite2-City.tar.gz"

    tar -xzf "$tmpdir/GeoLite2-City.tar.gz" -C "$tmpdir"

    mmdb="$(find "$tmpdir" -name 'GeoLite2-City.mmdb' | head -1)"
    if [ -z "$mmdb" ]; then
        echo "ERROR: GeoLite2-City.mmdb not found in downloaded archive" >&2
        exit 1
    fi

    mv "$mmdb" "$MMDB_PATH"
    echo "GeoLite2-City database saved to $MMDB_PATH"
}

if [ ! -f "$MMDB_PATH" ]; then
    if [ -z "${MAXMIND_LICENSE_KEY:-}" ]; then
        echo "WARNING: $MMDB_PATH not found and MAXMIND_LICENSE_KEY is not set; IP geolocation will be unavailable"
    else
        download_geolite2
    fi
else
    echo "GeoLite2-City database already present at $MMDB_PATH"
fi

exec "$@"
