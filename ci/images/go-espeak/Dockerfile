FROM alpine:latest as builder

RUN apk update && apk --no-cache add  \
    autoconf \
    automake \
    make \
    libtool \
    espeak \
    portaudio \
    pkgconfig \
    ca-certificates \
    g++ \
    gcc \
    git \
	&& rm -rf /var/lib/apt/lists/*

RUN git clone https://github.com/espeak-ng/espeak-ng.git \
    && cd espeak-ng \
    && ./autogen.sh \
    && ./configure \
        --prefix=/usr \
        --with-extdict-ru \
        --with-extdict-zh \
        --with-extdict-zhy \
    && make \
    && make install \
    && rm -rf /tmp/espeak

ENV GOLANG_VERSION 1.15.3
RUN mkdir /lib64 \
  && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
RUN url="https://golang.org/dl/go1.15.3.linux-amd64.tar.gz"; \
  wget -O go.tgz "$url"; \
  tar -C /usr/local -xzf go.tgz; \
  rm go.tgz; \
  export PATH="/usr/local/go/bin:$PATH"; \
  go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
WORKDIR "$GOPATH"
