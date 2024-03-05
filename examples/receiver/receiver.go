package main

import (
    "fmt"
    "time"
    "net"

    "github.com/patopesto/go-sacn"
)


func main() {
    fmt.Println("hello")

    itf, _ := net.InterfaceByName("en0")
    receiver := sacn.NewReceiver(itf)
    receiver.JoinUniverse(1)
    receiver.Start()

    for {
    	time.Sleep(1)
    }
}