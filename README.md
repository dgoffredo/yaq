yaq
===
**y**et **a**nother **q**ueue

<img alt="yaks in a queue" src="yaks.jpg" width="400"/>

Why
---
It's just for fun.

I want a queue kind of like what you would get with Redis's [LPUSH][1] and
[BRPOP][2] commands, and using an [append-only file][3] for persistence.

However, I want the queues to persist on the file system visible as directories
full of files.  That is, I want to use directory entries as the core storage
data structure for persistent message queue.  I want to be able to `cd` into
a queue and inspect its message files.  Not for any particular reason.

It will speak HTTP, handle a zillion concurrent connections, and be as simple
as I can manage.

What
----
`yaq` is an HTTP server, written in Go, that provides persistent message queues
backed by the file system.

<dl>
    <dt>GET {queue-path}</dt>
    <dt>GET {queue-path}?quantity={integer}</dt>
    <dt>GET {queue-path}?quantity=unlimited</dt>
    <dt>GET {queue-path}?timeout={time-spec}</dt>
    <dd>Here's my explanation blah blah.</dd>
    <dt>POST {queue-path}</dt>
    <dd>Here's another explanation blah blah.</dd>
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
/tmp/queues/
/tmp/queues/@admin.fifo
/tmp/queues/fish/
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
204
$ curl http://localhost:1337/fish/saltwater &
$ curl --request POST --data 'wrasse' http://localhost:1337/fish/saltwater
wrasse
[2]+  Done                    curl http://localhost:1337/fish/saltwater
$ find /tmp/queues/
/tmp/queues/
/tmp/queues/fish/
/tmp/queues/fish/saltwater/
/tmp/queues/fish/saltwater/@next
$ cat /tmp/queues/fish/saltwater/@next
@0000000003
$ echo exit >/tmp/queues/@admin.fifo
[1]+  Done                    ./yaq /tmp/queues 127.0.0.1:1337
$
```

More
----
TODO

[1]: https://redis.io/commands/lpush
[2]: https://redis.io/commands/brpop
[3]: https://redis.io/topics/persistence
