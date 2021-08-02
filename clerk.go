package yaq

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// TODO doc all the things
type clerk struct {
	// TODO: document
	Enqueue chan messageSend
	// TODO: document
	Dequeue chan messageReceive
	// TODO: document
	Registry *Registry
	// TODO: document
	Queue string
	// TODO: document
	QueueDirectory string
}

// TODO: document (and later replace some of these)
const emptyTimeout time.Duration = 10 * time.Second
const maxCapacity int = 3
const permissions os.FileMode = 0644
const messageFileNameFormat string = "@%010d"

// messageFileName returns the file name for a queue message having the
// specified messageNumber.  Message file names are the number, but
// additionally:
// - prefixed with a "@" to avoid collisions with queue names, and
// - padded with leading zeros so that lexicographical ordering is the same
//   as numerical ordering, at least up to a very large number.
func messageFileName(messageNumber int64) string {
	return fmt.Sprintf(messageFileNameFormat, messageNumber)
}

// TODO: refactor oldest() into something that can be used for both oldest() and next().

func (clerk *clerk) oldest() (result int64) {
	contents, err := os.ReadFile(filepath.Join(clerk.QueueDirectory, "@oldest"))
	if err == os.ErrNotExist {
		return 0 // TODO
	}
	if err != nil {
		// TODO: seems risky
		panic(err)
	}

	// Parse the number from the contents of the file.
	numParsed, err := fmt.Fscanf(bytes.NewReader(contents), messageFileNameFormat, &result)
	if err != nil {
		// TODO: seems risky
		panic(err)
	}
	if numParsed != 1 {
		// TODO: seems risky
		panic("unable to parse message number from @oldest")
	}

	return
}

// TODO: document
func (clerk *clerk) Run() {
	if err := os.MkdirAll(clerk.QueueDirectory, permissions); err != nil {
		// TODO: need a better policy
		panic(fmt.Sprintf("unable to create directory: %q error: %v", clerk.QueueDirectory, err))
	}
	// TODO: read/create queue's state
	// TODO: loop, with cases for:
	//     - empty (could also be full)
	//     - full (could also be empty)
	//     - neither
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
