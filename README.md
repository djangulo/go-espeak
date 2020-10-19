# GO-ESPEAK

Golang C bindings for the `espeak` voice synthesizer.

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
	//
	espeak.TextToSpeech(
		"Hello world!", // Text to speak
		nil,            // voice to use, nil == DefaultVoice (en-us male)
		"play",         // outfile to save to, "play" just plays the synth
		nil,            // Parameters for voice modulation, nil == DefaultParameters
	)
}

```