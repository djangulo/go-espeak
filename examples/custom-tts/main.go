package main

/*
#cgo CFLAGS: -I/usr/include/espeak
#cgo LDFLAGS: -lportaudio -lespeak
#include <stdio.h>
#include <string.h>
#include <malloc.h>
#include <speak_lib.h>


static inline void *userData(espeak_EVENT *event)  {
	if (event != NULL)
		if (event->user_data != NULL)
			return event->user_data;

	return NULL;
}

extern int mySynthCallback(short *wav, int numsamples , espeak_EVENT *events);
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/djangulo/go-espeak"
)

func main() {
	MyCustomTTS("Hello world!", "alice")
	MyCustomTTS("Hi there!", "bob")
	fmt.Printf("Done! Alice has %d samples, Bob has %d samples\n", len(alicesData), len(bobsData))
}

func MyCustomTTS(text, user string) {
	// this id is the address of the data as its acted on by the SynthCallback
	// function, its passed to the callback events.
	espeak.Init(espeak.Synchronous, 1024, nil, espeak.PhonemeEvents)
	espeak.SetVoiceByProps(espeak.DefaultVoice)
	espeak.NewParameters().SetVoiceParams()
	espeak.SetSynthCallback(C.mySynthCallback)
	// espeak internally feeds the id returned by Init to the user_data,
	// but you can also pass arbitrary objects
	espeak.Synth(text, espeak.CharsAuto, 0, 0, espeak.Character, nil, unsafe.Pointer(&user))
	espeak.Synchronize()

	// at this point, the data is populated, write it to a file, distort it or whatever
}

var (
	alicesData = make([]int16, 0)
	bobsData   = make([]int16, 0)
)

//export mySynthCallback
func mySynthCallback(wav *C.short, numsamples C.int, events *C.espeak_EVENT) C.int {
	if wav == nil {
		return 1
	}
	// we passed a *string in Synth, we have to unsafely cast it into a *string
	// to extract it. C.userData is defined in the header (safely dereferences
	// the *C.espeak_EVENT object).
	user := (*string)(unsafe.Pointer(C.userData(events)))
	length := int(numsamples)
	if *user == "alice" {
		alicesData = append(
			alicesData,
			(*[1 << 28]int16)(unsafe.Pointer(wav))[:length:length]...,
		)
		fmt.Printf("%s, you have %d samples so far\n", *user, len(alicesData))
	} else {
		bobsData = append(
			bobsData,
			(*[1 << 28]int16)(unsafe.Pointer(wav))[:length:length]...,
		)
		fmt.Printf("%s, you have %d samples so far\n", *user, len(bobsData))
	}
	return 0
}
