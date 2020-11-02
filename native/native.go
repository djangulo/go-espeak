// Copyright 2020 djangulo. All rights reserved. Use of this source code is
// governed by an MIT license that can be found in the LICENSE file.

//Package native has espeak native C implementation (called by Go)
// to synthesize audio or write to .wav.
package native

/*
#cgo CFLAGS: -I/usr/include/espeak
#cgo LDFLAGS: -l portaudio -l espeak
#include <stdio.h>
#include <string.h>
#include <malloc.h>
#include <speak_lib.h>
unsigned int samplestotal = 0;
int samplerate;
char *wavefile=NULL;
FILE *f_wavfile = NULL;
int OpenWavFile(char *path, int rate);
void CloseWavFile();
int callback(short *wav, int numsamples, espeak_EVENT *events);

int callback(short *wav, int numsamples, espeak_EVENT *events)
{
	int type;
	if(wav == NULL)
	{
		CloseWavFile();
		return(1);
	}
	if(f_wavfile == NULL){
		if(OpenWavFile(wavefile, samplerate) != 0){
			return(1);
		}
	}
	if(numsamples > 0){
		samplestotal += numsamples;
		fwrite(wav,numsamples*2,1,f_wavfile);
	}
	return(0);
}
////////////////////////////////////////////////////////////////////////////
// Static functions, sourced from espeak
////////////////////////////////////////////////////////////////////////////
// Write4Bytes: Write 4 bytes to a file, least significant first.
static void Write4Bytes(FILE *f, int value)
{
    int ix;
    for (ix = 0; ix < 4; ix++)
    {
        fputc(value & 0xff, f);
        value = value >> 8;
    }
}
int OpenWavFile(char *path, int rate)
{
    static unsigned char wave_hdr[44] = {
        'R', 'I', 'F', 'F', 0x24, 0xf0, 0xff, 0x7f, 'W', 'A', 'V', 'E', 'f', 'm', 't', ' ',
        0x10, 0, 0, 0, 1, 0, 1, 0, 9, 0x3d, 0, 0, 0x12, 0x7a, 0, 0,
        2, 0, 0x10, 0, 'd', 'a', 't', 'a', 0x00, 0xf0, 0xff, 0x7f};
    if (path == NULL)
        return (2);
    if (path[0] == 0)
        return (0);
    if (strcmp(path, "stdout") == 0)
        f_wavfile = stdout;
    else
        f_wavfile = fopen(path, "wb");
    if (f_wavfile == NULL)
    {
        fprintf(stderr, "Can't write to: '%s'\n", path);
        return (1);
    }
    fwrite(wave_hdr, 1, 24, f_wavfile);
    Write4Bytes(f_wavfile, rate);
    Write4Bytes(f_wavfile, rate * 2);
    fwrite(&wave_hdr[32], 1, 12, f_wavfile);
    return (0);
}
void CloseWavFile()
{
    unsigned int pos;
    if ((f_wavfile == NULL) || (f_wavfile == stdout))
        return;
    fflush(f_wavfile);
    pos = ftell(f_wavfile);
    fseek(f_wavfile, 4, SEEK_SET);
    Write4Bytes(f_wavfile, pos - 8);
    fseek(f_wavfile, 40, SEEK_SET);
    Write4Bytes(f_wavfile, pos - 44);
    fclose(f_wavfile);
    f_wavfile = NULL;
}
*/
import "C"
import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/djangulo/go-espeak"
)

// TextToSpeech reproduces text, using voice, modified by params.
// If params is nil, default parameters are used.
// If outfile is an empty string or "play", the audio is spoken to the system
// default's audio output; otherwise is appended with .wav and saved to
// params.Dir/outfile[.wav]. Returns the number of samples written to file,
// if any.
func TextToSpeech(text string, voice *espeak.Voice, outfile string, params *espeak.Parameters) (uint64, error) {
	if text == "" {
		return 0, espeak.ErrEmptyText
	}
	if params == nil {
		params = espeak.NewParameters()
	}
	if voice == nil {
		voice = espeak.DefaultVoice
	}

	var (
		uid      *C.uint
		options  C.int = C.espeakINITIALIZE_PHONEME_EVENTS
		flags    C.int = C.espeakCHARS_AUTO | C.espeakENDPAUSE
		userData unsafe.Pointer
		// bufLength length in mS of sound buffers passed to the SynthCallback
		// function. Value=0 gives a default of 200mS
		bufLength C.int = 200
		output    C.espeak_AUDIO_OUTPUT
		path      *C.char
	)

	if outfile == "" || outfile == "play" {
		output = C.AUDIO_OUTPUT_PLAYBACK
	} else {
		output = C.AUDIO_OUTPUT_SYNCHRONOUS
	}

	outfile = ensureWavSuffix(outfile)
	if err := os.MkdirAll(params.Dir, 0755); err != nil {
		return 0, err
	}
	outfile = filepath.Join(params.Dir, outfile)

	C.wavefile = C.CString(outfile)
	defer C.free(unsafe.Pointer(C.wavefile))

	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))

	C.samplerate = C.espeak_Initialize(output, bufLength, path, options)
	if C.samplerate == -1 {
		return 0, espeak.EErrInternal
	}

	if err := params.SetVoiceParams(); err != nil {
		return 0, err
	}

	//set call back
	C.espeak_SetSynthCallback((*C.t_espeak_callback)(C.callback))
	if err := espeak.SetVoiceByName(voice.Name); err != nil {
		return 0, err
	}

	ee := C.espeak_Synth(
		unsafe.Pointer(ctext),
		C.ulong(len(text)),
		C.uint(0),
		C.POS_CHARACTER,
		C.uint(0),
		C.uint(flags),
		uid,
		userData)
	if err := errFromCode(ee); err != nil {
		return 0, err
	}

	ee = C.espeak_Synchronize()
	if err := errFromCode(ee); err != nil {
		return 0, err
	}

	return uint64(C.samplestotal), nil
}

func ensureWavSuffix(s string) string {
	for s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	if !strings.HasSuffix(s, ".wav") {
		s += ".wav"
	}
	return s
}

func errFromCode(code C.espeak_ERROR) error {
	switch code {
	case C.EE_OK:
		return nil
	case C.EE_INTERNAL_ERROR:
		return espeak.EErrInternal
	case C.EE_BUFFER_FULL:
		return espeak.EErrBufferFull
	case C.EE_NOT_FOUND:
		return espeak.EErrNotFound
	default:
		return espeak.ErrUnknown
	}
}
