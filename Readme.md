# GOPOS

GOPOS is a library for managing POS printers, allowing you to print both text and images. You can connect via Bluetooth or any other protocol that suits you.

## Getting started

```shell
go get -u github.com/LimitR/gopos
```

## Running

```go
package main

import "github.com/LimitR/gopos"

func main() {
    printerClient, err := gopos.NewPrinter(gopos.OptionConnect{
	AddrPrinter:        "D7:37:1A:D0:CC:A8", 
	DomainConnect:      syscall.AF_BLUETOOTH, 
	TypeConnect:        syscall.SOCK_STREAM, 
	ProtoConnect:       unix.BTPROTO_RFCOMM, 
	PowerPrinter:       0xff,
	QualityPrinter:     0x05,
	DrawingModePrinter: 0x01,
    })

    if err != nil {
        panic(err)
    }

    err = printerClient.PrintText("Hello world", &gopos.OptionPrint{
        FontPath: gopos.DefaultFont,
        FontSize: 20,
    })
}
```