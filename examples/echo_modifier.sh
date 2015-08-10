#!/usr/bin/env bash
while read line; do
    decoded=$(echo "$line" | xxd -r -p)

    header=$(echo "$decoded" | head -n +1)
    payload=$(echo "$decoded" | tail -n +2)

    encoded=$(echo -e "$header\n$payload" | xxd -p | tr -d "\\n")

    >&2 echo ""
    >&2 echo "[DEBUG][MIDDLEWARE] ==================================="

    case ${header:0:1} in
    "2")
        >&2 echo "[DEBUG][MIDDLEWARE] Request type: Replayed Response"
        ;;
    "1")
        >&2 echo "[DEBUG][MIDDLEWARE] Request type: Request"
        echo "$encoded"
        ;;
    *)
        >&2 echo "[DEBUG][MIDDLEWARE] Unknown request type $header"
    esac
    >&2 echo "[DEBUG][MIDDLEWARE] ==================================="

    >&2 echo "[DEBUG][MIDDLEWARE] Original data: $line"
    >&2 echo "[DEBUG][MIDDLEWARE] Decoded request: $decoded"
    >&2 echo "[DEBUG][MIDDLEWARE] Encoded data: $encoded"
done;
