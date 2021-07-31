package yaq

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type refCountedClerk struct {
	Clerk    clerk
	RefCount uint
}

type Registry struct {
	// TODO
	RootDirectory string
	mutex         sync.Mutex
	queues        map[string]*refCountedClerk
}

// TODO
func (registry *Registry) withClerk(queue string, callback func(*clerk) error) error {
	registry.mutex.Lock()
	entry, found := registry.queues[queue]
	if !found {
		entry = &refCountedClerk{
			Clerk: clerk{
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

func (registry *Registry) Enqueue(queue string, request http.Request, timeout time.Duration) error {
	// Get the clerk that manages the queue, or create one.
	return registry.withClerk(queue, func(clerk *clerk) error {
		// Send the request to the clerk, and then wait for the receiver to notify
		// us that the message transfer is complete (via the Done chan).
		message := messageSend{
			Request: request,
			Done:    make(chan struct{}),
		}

		disconnects := request.Context().Done()

		select {
		case clerk.Enqueue <- message:
		case <-disconnects:
			return fmt.Errorf("Registry.Enqueue: the sender disconnected")
		}

		select {
		case <-message.Done:
		case <-disconnects:
			return fmt.Errorf("Registry.Enqueue: the sender disconnected")
		}

		return nil
	})
}

// copyHeader adds the HTTP header whose name is the specified `which`
// from the specified headers `from` to the specified headers `to`.
func copyHeader(which string, from http.Header, to http.Header) {
	for _, value := range from.Values(which) {
		to.Add(which, value)
	}
}

// TODO: tricky to figure out the signature...
func (registry *Registry) Dequeue(requestCtx context.Context, queue string, writer http.ResponseWriter, timeout time.Duration) error {
	return registry.withClerk(queue, func(clerk *clerk) error {
		var message messageReceive
		select {
		case message = <-clerk.Dequeue:
		case <-requestCtx.Done():
			return fmt.Errorf("Registry.Dequeue: the sender disconnected")
		}

		fmt.Println(message)

		// TODO: Forward content-type and content-encoding, if present.
		// TODO: The ioutil.Copy(....)
		// TODO: then message.Done

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
