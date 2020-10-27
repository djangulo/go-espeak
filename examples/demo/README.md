# Demo

This is a website demo for `go-espeak`.

There is a live version running at <a target="_blank" rel="noopener noreferrer" href="https://go-espeak-demo.djangulo.com">https://go-espeak-demo.djangulo.com</a>.

`.wav` files get created and saved under `static/audio` and played instantly. Any `.wav` older than 1 seconds gets removed.

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