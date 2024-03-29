//go:build linux
// +build linux

package buse

import (
	"net"
	"os"
	"syscall"

	"github.com/dop251/nbd"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Device struct {
	size       int64
	blockSize  int
	device     string
	driver     nbd.Driver
	deviceFp   *os.File
	sock       int
	client     *nbd.Client
	srvConn    *nbd.ServerConn
	disconnect chan struct{}
	log        *logrus.Logger
	readOnly   bool
}

func (bd *Device) startNBDClient() {
	bd.client = nbd.NewClient(bd.device, bd.sock, bd.size)
	if bd.log != nil {
		bd.client.SetLogger(bd.log)
	}

	if bd.blockSize > 0 {
		bd.client.SetBlockSize(bd.blockSize)
	} else {
		bd.client.SetBlockSize(512)
	}

	if _, ok := bd.driver.(nbd.Syncer); ok {
		bd.client.SetSendFlush(true)
	}
	if _, ok := bd.driver.(nbd.Trimmer); ok {
		bd.client.SetSendTrim(true)
	}

	bd.client.SetReadOnly(bd.readOnly)

	err := bd.client.Run()
	if err != nil && bd.log != nil {
		bd.log.Errorf("Client has failed: %v", err)
	}

	bd.srvConn.Close()
	bd.disconnect <- struct{}{}
}

// Disconnect disconnects the Device and interrupts the Run()
func (bd *Device) Disconnect() {
	if err := bd.client.Close(); err == nil {
		<-bd.disconnect
	}
}

func (bd *Device) SetMaxProc(p int) {
	bd.srvConn.SetMaxProc(p)
}

func (bd *Device) SetPool(pool nbd.ProcPool) {
	bd.srvConn.SetPool(pool)
}

func (bd *Device) SetLogger(log *logrus.Logger) {
	bd.log = log
	bd.srvConn.SetLogger(log)
}

func (bd *Device) SetBlockSize(size int) {
	bd.blockSize = size
}

func (bd *Device) SetReadOnly(flag bool) {
	bd.readOnly = flag
}

// Run connects a Device to an actual device file
// and starts handling requests. It does not return until it's done serving requests.
func (bd *Device) Run() error {
	go bd.startNBDClient()
	defer bd.srvConn.Close()
	return bd.srvConn.Serve()
}

func NewDevice(device string, size int64, driver nbd.Driver) (*Device, error) {
	buseDevice := &Device{size: size, device: device, driver: driver}
	sockPair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, errors.Wrap(err, "socketpair() failed")
	}
	buseDevice.deviceFp, err = os.OpenFile(device, os.O_RDWR, 0600)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot open '%s'", device)
	}
	fp := os.NewFile(uintptr(sockPair[0]), "unix")
	conn, err := net.FileConn(fp)
	if err != nil {
		return nil, err
	}
	fp.Close() // duplicate
	buseDevice.srvConn = nbd.NewServerConn(conn, driver)
	buseDevice.sock = sockPair[1]
	buseDevice.disconnect = make(chan struct{}, 1)
	return buseDevice, nil
}
