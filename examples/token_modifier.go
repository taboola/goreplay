package main

import (
    "os"
    "bufio"
    "encoding/hex"
    "github.com/buger/gor/proto"
    "bytes"
    "fmt"
)

// requestID -> originalToken
var originalTokens map[string][]byte

// originalToken -> replayedToken
var tokenAliases map[string][]byte

func main() {
    originalTokens = make(map[string][]byte)
    tokenAliases = make(map[string][]byte)

    scanner := bufio.NewScanner(os.Stdin)

    for scanner.Scan() {
        encoded := scanner.Bytes()
        buf := make([]byte, len(encoded)/2)
        hex.Decode(buf, encoded)

        go process(buf)
    }
}

func process(buf []byte) {
    // First byte indicate payload type, possible values:
    //  1 - Request
    //  2 - Response
    //  3 - ReplayedResponse
    payloadType := buf[0]
    headerSize := 42
    header := buf[:headerSize]
    // For each request you should receive 3 payloads (request, response, replayed response) with same request id
    reqID := string(header[2:headerSize])
    payload := buf[headerSize:]

    Debug("Received payload:", string(buf))

    switch payloadType {
    case '1':
        if bytes.Equal(proto.Path(payload), []byte("/token")) {
            originalTokens[reqID] = []byte{}
            Debug("Found token request:", reqID)
        } else {
            tokenVal, vs, _ := proto.PathParam(payload, []byte("token"))

            if vs != -1 { // If there is GET token param
                if alias, ok := tokenAliases[string(tokenVal)]; ok {
                    // Rewrite original token to alias
                    payload = proto.SetPathParam(payload, []byte("token"), alias)

                    // Copy modified payload to our buffer
                    copy(buf[headerSize:], payload)
                }
            }
        }

        // Re-compute length in case if payload was modified
        bufLen := len(header) + len(payload)
        // Encoding request and sending it back
        dst := make([]byte, bufLen*2+1)
        hex.Encode(dst, buf[:bufLen])
        dst[len(dst)-1] = '\n'

        os.Stdout.Write(dst)

        return
    case '2': // Original response
        if _, ok := originalTokens[reqID]; ok {
            // Token is inside response body
            secureToken := proto.Body(payload)
            originalTokens[reqID] = secureToken
            Debug("Remember origial token:", string(secureToken))
        }
    case '3': // Replayed response
        if originalToken, ok := originalTokens[reqID]; ok {
            delete(originalTokens, reqID)
            secureToken := proto.Body(payload)
            tokenAliases[string(originalToken)] = secureToken

            Debug("Create alias for new token token, was:", string(originalToken), "now:", string(secureToken))
        }
    }
}

func Debug(args ...interface{}) {
    fmt.Fprint(os.Stderr, "[DEBUG][TOKEN-MOD] ")
    fmt.Fprintln(os.Stderr, args...)
}