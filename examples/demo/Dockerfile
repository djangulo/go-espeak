FROM djangulo/go-espeak:latest
LABEL maintainer="Denis Angulo <djal@tuta.io>"
WORKDIR /app

RUN git clone https://github.com/djangulo/go-espeak.git /tmp/go-espeak \
    && cp -r /tmp/go-espeak/examples/demo/* . \
    && rm -rf /tmp/go-espeak
RUN go get -u -v github.com/djangulo/go-espeak
RUN CGO_ENABLED=1 GOOS=linux go build -p 1 -a -o ./go-espeak-demo ./...

RUN addgroup golang \
    && adduser --ingroup golang golang
RUN mkdir -p ./static/audio \
    && mkdir -p ./static/downloads \
    && chown -R golang:golang ./static \
    && chown golang:golang ./index.html
RUN chmod +x ./go-espeak-demo \
    && chown golang:golang ./go-espeak-demo

USER golang
