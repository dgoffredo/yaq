package yaq

import (
	"io"
	"net/http"
)

type messageSend struct {
	Request http.Request
	Done    chan struct{}
	// TODO
}

type messageReceive interface {
	io.Reader
	Headers() map[string][]string
	Done()
}
