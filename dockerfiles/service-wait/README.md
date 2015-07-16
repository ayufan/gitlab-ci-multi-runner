`gitlab/runner-wait` is a really small Docker utility that blocks until another container is accepting TCP connections.

Use it like this:

    $ docker run -d --name mycontainer some-image-or-other
    $ docker run --link mycontainer:mycontainer aanand/wait
    waiting for TCP connection to 172.17.0.105:5432......ok

Just link a single container to it - it doesn't matter what the link alias is.
