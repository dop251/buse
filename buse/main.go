package main

import (
	"io"
	"log"
	"os"

	"github.com/dop251/buse"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("Usage buse <file> </dev/nbd...>")
	}

	f, err := os.OpenFile(os.Args[1], os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Could not open file: %v", err)
	}

	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		log.Fatalf("Could not seek: %v", err)
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatalf("Could not seek: %v", err)
	}

	dev, err := buse.NewDevice(os.Args[2], size, f)
	if err != nil {
		log.Fatalf("Could not create buse device: %v", err)
	}

	dev.SetMaxProc(4)

	defer dev.Disconnect()

	dev.Run()
}
