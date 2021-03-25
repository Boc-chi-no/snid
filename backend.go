package main

import (
	"net"
)

type BackendConn interface {
	net.Conn
	CloseWrite() error
}

type BackendDialer interface {
	Dial(string) (BackendConn, error)
}
