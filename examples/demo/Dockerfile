FROM golang:1.15 as builder
LABEL maintainer="Denis Angulo <djal@tuta.io>"
WORKDIR $GOPATH/src/github.com/djangulo/go-espeak/examples/demo

RUN go get -v ./...
RUN CGO_ENABLED=1 GOOS=linux go build -p 1 -a -o $GOPATH/bin/go-espeak-demo ./...

FROM alpine:latest
RUN addgroup -S golang \
    && adduser -S -G golang golang
RUN apk update \
    && apk --no-cache add ca-certificates
COPY --from=builder /go/bin/go-espeak-demo /go-espeak-demo
COPY --from=builder /go/src/github.com/djangulo/go-espeak/examples/demo/index.html /index.html
RUN mkdir -p /static/audio \
    && chown -R golang:golang /static \
    && chown golang:golang /index.html
RUN chmod +x /go-espeak-demo \
    && chown golang:golang /go-espeak-demo

USER golang
