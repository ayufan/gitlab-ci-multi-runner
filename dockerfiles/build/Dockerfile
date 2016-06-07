FROM alpine

RUN apk add --update bash wget ca-certificates openssl git tar

RUN wget -O /tmp/glibc.apk "https://circle-artifacts.com/gh/andyshinn/alpine-pkg-glibc/6/artifacts/0/home/ubuntu/alpine-pkg-glibc/packages/x86_64/glibc-2.21-r2.apk" && \
    apk add --allow-untrusted /tmp/glibc.apk && \
 	/usr/glibc/usr/bin/ldconfig /lib /usr/glibc/usr/lib && \
 	rm /tmp/glibc.apk

COPY ./ /usr/bin/
ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]
