#!/usr/bin/env bash
while read line; do
    decoded=$(echo -e "$line" | xxd -r -p)

    header=$(echo -e "$decoded" | head -n +1)
    payload=$(echo -e "$decoded" | tail -n +2)

    encoded=$(echo -e "$header\n$payload" | xxd -p | tr -d "\\n")

    >&2 echo ""
    >&2 echo "[DEBUG][ECHO] ==================================="

    case ${header:0:1} in
    "3")
        >&2 echo "[DEBUG][ECHO] Request type: Original Response"
        ;;
    "2")
        >&2 echo "[DEBUG][ECHO] Request type: Replayed Response"
        ;;
    "1")
        >&2 echo "[DEBUG][ECHO] Request type: Request"
        echo "$encoded"
        ;;
    *)
        >&2 echo "[DEBUG][ECHO] Unknown request type $header"
    esac
    >&2 echo "[DEBUG][ECHO] ==================================="

    >&2 echo "[DEBUG][ECHO] Original data: $line"
    >&2 echo "[DEBUG][ECHO] Decoded request: $decoded"
    >&2 echo "[DEBUG][ECHO] Encoded data: $encoded"
done;
