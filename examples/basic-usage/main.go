// Copyright 2020 djangulo. All rights reserved. Use of this source code is
// governed by an MIT license that can be found in the LICENSE file.
package main

import (
	"fmt"

	"github.com/djangulo/go-espeak"
)

func main() {

	// need to call terminate so espeak can clean itself out
	defer espeak.Terminate()
	params := espeak.NewParameters().WithDir(".")
	var written uint64
	written, _ = espeak.TextToSpeech(
		"Hello World!", // Text to speak
		nil,            // voice to use, nil == DefaultVoice (en-us male)
		"hello.wav",    // if "" or "play", it plays to default audio out
		params,         // Parameters for voice modulation, nil == DefaultParameters
	)
	fmt.Printf("bytes written to hello.wav:\t%d\n", written)

	// get a random spanish voice
	v, _ := espeak.VoiceFromSpec(&espeak.Voice{Languages: "es"})
	written, _ = espeak.TextToSpeech("¡Hola mundo!", v, "hola.wav", params)
	fmt.Printf("bytes written to hola.wav:\t%d\n", written)
}
