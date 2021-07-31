- "garbage collect" queue goroutines?
- "garbage collect" empty queues on the file system?
- queue configuration
    - maximum capacity in messages
    - maximum total storage (files + directories)
    - pattern language?

Maybe something like this:

    {regex1}
        constraint1
        constraint2
        ...

    {regex2}
    ...

e.g.

    .*
        messageCount < 1024*1024
        totalStorageBytes < 64*1024*1024*1024

    /large/.*
        totalStorageBytes < 1024*1024*1024
        
    .*/bill
        messageCount < 1

Let's make the initial cut of this project be without configuration, without
timeouts, without streaming, etc.
