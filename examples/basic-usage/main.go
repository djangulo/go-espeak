package main

import (
	"github.com/djangulo/go-espeak"
)

func main() {

	defer espeak.Terminate()
	espeak.TextToSpeech(
		"Hello world", // Text to speak
		nil,           // voice to use, nil == DefaultVoice (en-us male)
		"play",        // outfile to save to, "play" just plays the synth
		nil,           // Parameters for voice modulation, nil == DefaultParameters
	)
	// get a random spanish voice
	v, _ := espeak.VoiceFromSpec(&espeak.Voice{Languages: "es"})
	espeak.TextToSpeech("¡Hola mundo!", v, "play", nil)
}
