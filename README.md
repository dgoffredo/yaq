yaq
===
**y**et **a**nother **q**ueue

<img alt="yaks in a queue" src="yaks.webp" width="400"/>

Why
---
It's just for fun.

I want a queue kind of like what you would get with Redis's [LPUSH][1] and
[BRPOP][2] commands, and using an [append-only file][3] for persistence.

However, I want the queues to persist as directories full of files.  That is, I
want to use directory entries as the core storage data structure for persistent
message queue.  I want to be able to `cd` into a queue and inspect its message
files.  Not for any particular reason.

It will speak HTTP, handle a zillion concurrent connections, and be as simple
as I can manage.

What
----
`yaq` is an HTTP server, written in Go, that provides persistent message queues
backed by the file system.

### The REST API
Each queue has a name that is a URL path, e.g. `/profile-updates/v2/user12345`
is a queue, and so are `/profile-updates/v2/user43554`, `/`, and
`/snazzy%20party`.

<dl>
    <dt>GET {queue-path}</dt>
    <dt>GET {queue-path}?quantity={integer}</dt>
    <dt>GET {queue-path}?quantity=unlimited</dt>
    <dt>GET {queue-path}?timeout={time-spec}</dt>
    <dd>Dequeue the optionally specified quantity of message from the specified
    queue. If quantity is not specified, dequeue one message.  If quantity is
    "unlimited," then keep dequeueing messages until the client disconnects.
    If timeout is specified, then its value is a time duration as accepted by
    Go's <a href="https://pkg.go.dev/time#ParseDuration">time.ParseDuration</a>
    function.  If timeout is not specified, then it is infinite.  The timeout
    begins whenever the client is waiting for a message but the queue is empty.
    The timeout applies separately to each message requested (i.e. it resets
    when a message is received).  If only one message is being dequeued, then
    the body of the response is the message.  If multiple messages are being
    dequeued, then the body of the response is a <a href="https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Transfer-Encoding#chunked_encoding">chunked</a>
    <code>multipart/mixed</code> MIME message as described in <a href="https://datatracker.ietf.org/doc/html/rfc2046#section-5.1.1">RFC 2046 Section
    5.1.1 "Common Syntax"</a>.  If fewer than the specified quantity of
    messages are dequeued, such as when a timeout occurs or some other error,
    then the response will end with a <a href="https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Trailer">trailer</a> containing a
    <code>X-Next-Message-Status</code> header whose value is the effective HTTP
    response status code of the message that failed to dequeue.
    Some clients might prefer to avoid the "multipart" complexity by dequeuing
    messages one at a time using separate requests instead.
    <dd>The response will have one of the following status codes:
      <dl>
        <dt>200 OK</dt>
        <dd>One message was requested and successfully dequeued.  The response body is the message.</dd>
        <dt>202 Accepted</dt> 
        <dd>Two or more messages were requested and will be streamed in the response.  The response uses the
        <a href="https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Transfer-Encoding#chunked_encoding">chunked</a> <code>Transfer-Encoding</code> and is a <code>multipart/mixed</code> MIME message
        containing the dequeued messages.</dd>
        <dt>504 Gateway Timeout</dt>
        <dd>The timeout that was specified as a query parameter in the request expired before a message could be dequeued.</dd>
        <dt>500 Internal Server Error</dt>
        <dd>An error occurred.  The response body will be a description of the error.</dd>
      </dl>
    </dd>
    <dt>POST {queue-path}</dt>
    <dt>POST {queue-path}?timeout={time-spec}</dt>
    <dd>Enqueue a message onto the specified queue.  The body of the request is the
    message.  The request's <code>Content-Type</code> and
    <code>Content-Encoding</code> will remain associated with the message.
    If timeout is specified, then its value is a time duration as accepted by
    Go's <a href="https://pkg.go.dev/time#ParseDuration">time.ParseDuration</a>
    function.  If timeout is not specified, then it is infinite.  If a queue
    has reached its capacity, then attempts to enqueue messages will block until enough
    messages are dequeued to bring the queue back to below capacity.</dd>
    <dd>The response will have one of the following status codes:
      <dl>
        <dt>200 OK</dt>
        <dd>The message was successfully enqueued.</dd>
        <dt>504 Gateway Timeout</dt>
        <dd>The timeout that was specified as a query parameter in the request expired before the message could be enqueued.</dd>
        <dt>500 Internal Server Error</dt>
        <dd>An error occurred.  The response body will be a description of the error.</dd>
      </dl>
    </dd>
</dl>

How
---
```console
$ go build
$ mkdir /tmp/queues
$ ./yaq /tmp/queues 127.0.0.1:1337 &
Listening on 127.0.0.1:1337.  Storing queues under "/tmp/queues".
$ curl --request POST --data 'tuna' http://localhost:1337/fish/saltwater
$ curl --request POST --data 'swordfish' http://localhost:1337/fish/saltwater
$ find /tmp/queues
/tmp/queues
/tmp/queues/fish
/tmp/queues/fish/saltwater/
/tmp/queues/fish/saltwater/@0000000000
/tmp/queues/fish/saltwater/@0000000001
/tmp/queues/fish/saltwater/@next
/tmp/queues/fish/saltwater/@oldest
$ cat /tmp/queues/fish/saltwater/@0000000001
Content-Type: text/plain; charset=UTF-8

swordfish
$ cat /tmp/queues/fish/saltwater/@next
@0000000002
$ cat /tmp/queues/fish/saltwater/@oldest
@0000000000
$ curl http://localhost:1337/fish/saltwater
tuna
$ cat /tmp/queues/fish/saltwater/@oldest
@0000000001
$ curl http://localhost:1337/fish/saltwater
swordfish
$ curl http://localhost:1337/fish/saltwater
^C
$ curl http://localhost:1337/fish/saltwater?timeout=2s
$ curl --write-out '%{http_code}' http://localhost:1337/fish/saltwater?timeout=2s
504
$ curl http://localhost:1337/fish/saltwater | sed 's/^/from background consumer: /' &
$ curl --request POST --data 'wrasse' http://localhost:1337/fish/saltwater
from background consumer: wrasse
[2]+  Done                    curl http://localhost:1337/fish/saltwater | sed 's/^/from background consumer: /'
$ curl --request POST --data 'shark' http://localhost:1337/fish/saltwater
$ find /tmp/queues/
/tmp/queues/
/tmp/queues/fish/
/tmp/queues/fish/saltwater/
/tmp/queues/fish/saltwater/@0000000003
/tmp/queues/fish/saltwater/@next
$ cat /tmp/queues/fish/saltwater/@0000000003
Content-Type: text/plain; charset=UTF-8

shark
$ cat /tmp/queues/fish/saltwater/@next
@0000000004
$ kill %1
[1]+  Terminated              ./yaq /tmp/queues 127.0.0.1:1337
$ find /tmp/queues/
/tmp/queues/
/tmp/queues/fish/
/tmp/queues/fish/saltwater/
/tmp/queues/fish/saltwater/@0000000003
/tmp/queues/fish/saltwater/@next
$
```

[1]: https://redis.io/commands/lpush
[2]: https://redis.io/commands/brpop
[3]: https://redis.io/topics/persistence
