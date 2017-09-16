# Linux userspace virtual block device in Go

The idea is similar to FUSE but works for block devices. It uses [nbd](https://github.com/dop251/nbd)
over a unix socket.

## Usage scenario

```go
package main

import (
    "github.com/dop251/buse"
    "github.com/dop251/nbd"
)

// Implement your custom driver as nbd.Driver.
var driver nbd.Driver = ....

dev := buse.NewDevice("/dev/nbd0", size, driver)
dev.Run()
// The device will become available as /dev/nbd0

```
