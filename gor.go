package main

import (
    "flag"
    "fmt"
    "github.com/buger/gor/listener"
    "github.com/buger/gor/replay"
    "os"
)

var mode string

func main() {
    if len(os.Args) > 1 {
        mode = os.Args[1]
    }

    if mode != "listen" && mode != "replay" {
        fmt.Println("Usage: \n\tgor listen -h\n\tgor replay -h")
        return
    }
    
    // Remove mode attr
    os.Args = append(os.Args[:1], os.Args[2:]...)

    flag.Parse()

    switch mode {
    case "listen":
        listener.Run()
    case "replay":
        replay.Run()    
    }

}
