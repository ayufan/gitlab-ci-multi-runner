FROM alpine

RUN apk add --update bash wget ca-certificates openssl git

RUN wget -O /tmp/glibc.apk "https://circle-artifacts.com/gh/andyshinn/alpine-pkg-glibc/6/artifacts/0/home/ubuntu/alpine-pkg-glibc/packages/x86_64/glibc-2.21-r2.apk" && \
    apk add --allow-untrusted /tmp/glibc.apk && \
 	/usr/glibc/usr/bin/ldconfig /lib /usr/glibc/usr/lib && \
 	rm /tmp/glibc.apk

RUN wget -O /usr/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-linux-amd64 && \
	chmod +x /usr/bin/gitlab-ci-multi-runner && \
	ln -s /usr/bin/gitlab-ci-multi-runner /usr/bin/gitlab-runner

ENTRYPOINT ["/bin/bash"]
