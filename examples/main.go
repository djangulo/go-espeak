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
