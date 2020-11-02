// Copyright 2020 djangulo. All rights reserved. Use of this source code is
// governed by an MIT license that can be found in the LICENSE file..
package espeak_test

import "github.com/djangulo/go-espeak"

func ExampleTextToSpeech() {
	espeak.TextToSpeech("Hello world!", espeak.DefaultVoice, "play", nil)
	// or set an outfile name to save it
	// TextToSpeech("Hello world!", ENUSFemale, "hello-world.wav", nil)
}

// ExampleTextToSpeech_second show usage with a non-default voice.
func ExampleTextToSpeech_customVoice() {
	// output of
	//     ~$ espeak --voices=el
	//     Pty Language Age/Gender VoiceName          File          Other Languages
	//     5  el             M  greek                europe/el
	//     7  el             M  greek-mbrola-1       mb/mb-gr2
	greek := espeak.Voice{
		Languages:  "el",
		Gender:     espeak.Male,
		Name:       "greek",
		Identifier: "europe/el",
	}
	espeak.TextToSpeech("Γειά σου Κόσμε!", &greek, "play", nil)
}
