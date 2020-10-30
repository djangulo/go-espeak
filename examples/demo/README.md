# Demo

This is a website demo for `go-espeak`.

There is a live version running at <a target="_blank" rel="noopener noreferrer" href="https://go-espeak-demo.djangulo.com">https://go-espeak-demo.djangulo.com</a>.

`.wav` files get created and saved under directory `audio-dir` (default `static/audio`) and played instantly, if files are meant to be downloaded, they're saved under `downloads-dir` (default `static/downloads`). `.wav` files live for 0.5 seconds under `audio-dir`, 2 seconds under `downloads-dir`.

## Usage

Clone the main repo

```bash
~$ git clone https://github.com/djangulo/go-espeak.git
```

Change dir and run the demo.

```bash
~$ cd go-espeak/examples/demo
~$ go run ./...
```

Alternatively, you can specify which port to run at with `-p` (or `-port`)

```bash
~$ go run ./... -p 54321
```