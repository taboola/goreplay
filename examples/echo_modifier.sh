#!/usr/bin/env bash
while read line; do
    decoded=$(echo "$line" | xxd -r -p)
    encoded=$(echo "$decoded" | xxd -p | tr -d "\\n")
    echo "$encoded"

    >&2 echo "[DEBUG][MIDDLEWARE] Original data: $line"
    >&2 echo "[DEBUG][MIDDLEWARE] Decoded request: $decoded"
    >&2 echo "[DEBUG][MIDDLEWARE] Encoded data: $encoded"
done;
