FROM djangulo/go-espeak:latest
LABEL maintainer="Denis Angulo <djal@tuta.io>"
WORKDIR /app

RUN go get -v github.com/djangulo/go-espeak \
    && cp -r $GOPATH/src/github.com/djangulo/go-espeak/examples/demo/* .
RUN CGO_ENABLED=1 GOOS=linux go build -p 1 -a -o ./demo ./...

RUN addgroup golang \
    && adduser --ingroup golang golang
RUN mkdir -p ./static/audio \
    && chown -R golang:golang ./static \
    && chown golang:golang ./index.html
RUN chmod +x ./demo \
    && chown golang:golang ./demo

USER golang
