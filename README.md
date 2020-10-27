<a href="https://pkg.go.dev/github.com/djangulo/go-espeak"><img src="https://pkg.go.dev/badge/github.com/djangulo/go-espeak" alt="PkgGoDev"></a>
<a href="https://ci.djangulo.com/teams/djangulo/pipelines/go-espeak"><img src="https://ci.djangulo.com/api/v1/teams/djangulo/pipelines/go-espeak/jobs/test-unit/badge" alt="CI status"></a>

# go-espeak

Golang C bindings for the `espeak` voice synthesizer.

There is a live demo of its usage in the [examples/demo]

## Requirements

- Go >= 1.14 with `cgo` support
- <a target="_blank" rel="noopener noreferrer" href="http://espeak.sourceforge.net/">`espeak`</a>.

## Install

### Requirements

Arch

```bash
~# pacman -S espeak
```

Ubuntu

```bash
~# apt-get install espeak
```

Install go-espeak

```bash
~$ go get -u github.com/djangulo/go-espeak
```

## Usage

```golang
package main

import "github.com/djangulo/go-espeak"

func main() {
	espeak.TextToSpeech(
		"Hello world!", // Text to speak
		nil,            // voice to use, nil == DefaultVoice (en-us male)
		"play",         // outfile to save to, "play" just plays the synth
		nil,            // Parameters for voice modulation, nil == DefaultParameters
	)
}

```
