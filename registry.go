package yaq

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TODO: document
type refCountedClerk struct {
	Clerk    clerk
	RefCount uint
}

// TODO: document
type Registry struct {
	// TODO: document
	RootDirectory string
	// TODO: document
	mutex sync.Mutex
	// TODO: document
	queues map[string]*refCountedClerk
}

// withClerk invokes the specified callback with a clerk that manages the
// specified queue.  If there is no such clerk, withClerk first creates one.
// The clerk's reference count is incremented for the duration of the callback.
// withClerk returns whatever the callback returns.
func (registry *Registry) withClerk(queue string, callback func(*clerk) error) error {
	registry.mutex.Lock()
	entry, found := registry.queues[queue]
	if !found {
		entry = &refCountedClerk{
			Clerk: clerk{
				// TODO: context
				Enqueue:  make(chan messageSend),
				Dequeue:  make(chan messageReceive),
				Registry: registry,
				Queue:    queue,
			},
			RefCount: 0,
		}
		go entry.Clerk.Run()
		registry.queues[queue] = entry
	}
	entry.RefCount++
	registry.mutex.Unlock()

	result := callback(&entry.Clerk)

	registry.mutex.Lock()
	entry.RefCount--
	registry.mutex.Unlock()

	return result
}

// optionalTimeout returns a channel on which the current time will be sent
// when the specified deadline arrives.  Or, if deadline is nil, optionalTimeout
// returns a nil channel (receives will block forever, i.e. no timeout).
func optionalTimeout(deadline *time.Time) <-chan time.Time {
	var timeout <-chan time.Time
	if deadline != nil {
		timeout = time.NewTimer(time.Until(*deadline)).C
	}
	return timeout
}

// ErrTimeout is returned by Enqueue and Dequeue when the caller-specified
// deadline arrives before the message was sent or received, respectively.
var ErrTimeout = errors.New("timeout")

// TODO: document
func (registry *Registry) Enqueue(queue string, request http.Request, deadline *time.Time) error {
	// Get the clerk that manages the queue, or create one.
	return registry.withClerk(queue, func(clerk *clerk) error {
		// Send the request to the clerk, and then wait for the receiver to notify
		// us that the message transfer is complete (via the Done chan).
		message := messageSend{
			Request:  request,
			DoneChan: make(chan error),
		}

		disconnect := request.Context().Done()
		timeout := optionalTimeout(deadline)

		select {
		case clerk.Enqueue <- message:
		case <-disconnect:
			return fmt.Errorf("Registry.Enqueue: the sender disconnected")
		case <-timeout:
			return ErrTimeout
		}

		// Note that the timeout doesn't apply here, because we already "sent"
		// message, we're just waiting for its body to be read.
		select {
		case err := <-message.DoneChan:
			return err
		case <-disconnect:
			return fmt.Errorf("Registry.Enqueue: the sender disconnected")
		}
	})
}

// copyHeader adds the HTTP header whose name is the specified `which`
// from the specified headers `from` to the specified headers `to`.
func copyHeader(which string, from http.Header, to http.Header) {
	for _, value := range from.Values(which) {
		to.Add(which, value)
	}
}

// TODO: document
func (registry *Registry) Dequeue(requestCtx context.Context, queue string, writer http.ResponseWriter, deadline *time.Time) (err error) {
	return registry.withClerk(queue, func(clerk *clerk) error {
		var message messageReceive
		timeout := optionalTimeout(deadline)

		select {
		case message = <-clerk.Dequeue:
		case <-requestCtx.Done():
			return fmt.Errorf("Registry.Dequeue: the receiver disconnected")
		case <-timeout:
			return ErrTimeout
		}

		// If we're ultimately successful, then send nil to the message source
		// to notify them that we're done.  If we're not successful, then send
		// the error to the message source.
		defer func() { message.Done(err) }()

		// Forward Content-Type and Content-Encoding headers, if present.
		// Then copy the body of the message into the body of the response.
		copyHeader("Content-Type", message.Header(), writer.Header())
		copyHeader("Content-Encoding", message.Header(), writer.Header())
		if _, err := io.CopyN(writer, message.Body(), message.ContentLength()); err != nil {
			return err
		}

		return nil
	})
}

// RemoveIfUnused deletes the entry for the specified queue in this registry
// if its reference count is zero.  Return whether the entry has been removed,
// and any error that occurred.
func (registry *Registry) RemoveIfUnused(queue string) (bool, error) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	entry, found := registry.queues[queue]
	if !found {
		// "Not found" is unexpected, since I expect only the clerk itself
		// would be trying to remove its queue (so why, then, is it already
		// missing?).
		return true, fmt.Errorf("no entry for queue %q", queue)
	}

	if entry.RefCount > 0 {
		// Someone is using the queue.  Return `false`, meaning "not removed."
		return false, nil
	}

	// Nobody is using the queue.  Remove it.
	delete(registry.queues, queue)
	// Return `true`, meaning "removed."
	return true, nil
}
