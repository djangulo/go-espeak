# Custom TTS

This example shows how to create your own TTS using `go-espeak` utilities.

You can pass arbitrary data into the callback function through the user_data property of the *espeak_EVENT when you call `espeak.Synth`.
