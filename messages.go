package yaq

import (
	"io"
	"net/http"
)

// TODO: document
type messageSend struct {
	// TODO: document
	Request http.Request
	// TODO: document
	DoneChan chan error
	// TODO
}

// TODO: document
type messageReceive interface {
	// TODO: document
	Body() io.Reader
	// TODO: document
	Header() http.Header
	// TODO: document
	ContentLength() int64
	// TODO: document
	Done(error)
}
