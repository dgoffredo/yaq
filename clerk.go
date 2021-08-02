package yaq

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"time"
)

// TODO doc all the things
type clerk struct {
	// TODO: context
	Enqueue        chan messageSend
	Dequeue        chan messageReceive
	Registry       *Registry
	Queue          string
	RootDirectory  string
	queueDirectory string
}

// TODO: document
const emptyTimeout time.Duration = 10 * time.Second

// TODO: document
func (clerk *clerk) Run() {
	clerk.queueDirectory = "TODO"
	// TODO
}

// messageSend satisfies the messageReceive interface.

// TODO: document
func (message *messageSend) Body() io.Reader {
	return message.Request.Body
}

// TODO: document
func (message *messageSend) Header() http.Header {
	return message.Request.Header
}

// TODO: document
func (message *messageSend) ContentLength() int64 {
	return message.Request.ContentLength
}

// TODO: document
func (message *messageSend) Done(err error) {
	message.DoneChan <- err
}

// TODO: document (and all the fields)
type messageFromFile struct {
	File          os.File
	Path          string
	header        http.Header
	contentLength int64
}

// messageFromFile satisfies the messageReceive interface.

// TODO: document
func (message *messageFromFile) Body() io.Reader {
	return &message.File
}

// TODO: document
func (message *messageFromFile) Header() http.Header {
	if message.header == nil {
		message.loadHeader()
	}
	return message.header
}

// TODO: document
func (message *messageFromFile) ContentLength() int64 {
	if message.header == nil {
		message.loadHeader()
	}
	return message.contentLength
}

// loadHeader reads HTTP-style headers from the top of the file, and stores
// them in the header field.  It then parses other fields from the resulting
// header.
func (message *messageFromFile) loadHeader() {
	reader := textproto.NewReader(bufio.NewReader(&message.File))
	header, err := reader.ReadMIMEHeader()
	if err != nil {
		panic(fmt.Sprintf("failed to parse MIME (HTTP) headers in stored message: %v", err))
	}
	message.header = http.Header(header)

	message.parseContentLength()
}

// parseContentLength is called by loadHeader.  It populates contentLength
// with an integer parsed from the value of the Content-Length header.
func (message *messageFromFile) parseContentLength() {
	values := message.header.Values("Content-Length")
	if len(values) != 1 {
		panic(fmt.Sprintf("invalid Content-Length header: %v", values))
	}
	value := message.header.Values("Content-Length")[0]
	length, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid Content-Length header value: %q", value))
	}
	message.contentLength = length
}
