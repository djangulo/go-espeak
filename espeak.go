// Copyright 2020 djangulo. All rights reserved.
// Use of this source code is governed by a GPLv3
// license that can be found in the LICENSE file.

//Package espeak implements C bindings for the Espeak voice synthesizer.
package espeak

/*
#cgo CFLAGS: -I/usr/include/espeak
#cgo LDFLAGS: -lportaudio -lespeak
#include <stdio.h>
#include <string.h>
#include <malloc.h>
#include <speak_lib.h>
void* user_data;
unsigned int samplestotal = 0;
int samplerate;
char *wavefile=NULL;
FILE *f_wavfile = NULL;
const char *WordToString(unsigned int word);
static void Write4Bytes(FILE *f, int value);
int OpenWavFile(char *path, int rate);
static void CloseWavFile();
int SynthCallback(short *wav, int numsamples, espeak_EVENT *events);

int SynthCallback(short *wav, int numsamples, espeak_EVENT *events)
 {
	 int type;
	 if(wav == NULL)
	 {
		 CloseWavFile();
		 return(0);
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
 const char *WordToString(unsigned int word)
 {//========================================
 // Convert a phoneme mnemonic word into a string
	 int ix;
	 static char buf[5];

	 for(ix=0; ix<3; ix++){
		 buf[ix] = word >> (ix*8);
	 }
	 buf[4] = 0;
	 return(buf);
 }
 static void Write4Bytes(FILE *f, int value)
 {//=================================
 // Write 4 bytes to a file, least significant first
	 int ix;

	 for(ix=0; ix<4; ix++)
	 {
		 fputc(value & 0xff,f);
		 value = value >> 8;
	 }
 }
 int OpenWavFile(char *path, int rate)
 //===================================
 {
	 static unsigned char wave_hdr[44] = {
		 'R','I','F','F',0x24,0xf0,0xff,0x7f,'W','A','V','E','f','m','t',' ',
		 0x10,0,0,0,1,0,1,0,  9,0x3d,0,0,0x12,0x7a,0,0,
		 2,0,0x10,0,'d','a','t','a',  0x00,0xf0,0xff,0x7f};

	 if(path == NULL)
		 return(2);

	 if(path[0] == 0)
		 return(0);

	 if(strcmp(path,"stdout")==0)
		 f_wavfile = stdout;
	 else
		 f_wavfile = fopen(path,"wb");

	 if(f_wavfile == NULL)
	 {
		 fprintf(stderr,"Can't write to: '%s'\n",path);
		 return(1);
	 }


	 fwrite(wave_hdr,1,24,f_wavfile);
	 Write4Bytes(f_wavfile,rate);
	 Write4Bytes(f_wavfile,rate * 2);
	 fwrite(&wave_hdr[32],1,12,f_wavfile);
	 return(0);
 }   //  end of OpenWavFile
 static void CloseWavFile()
 //========================
 {
	 unsigned int pos;

	 if((f_wavfile==NULL) || (f_wavfile == stdout))
		 return;

	 fflush(f_wavfile);
	 pos = ftell(f_wavfile);

	 fseek(f_wavfile,4,SEEK_SET);
	 Write4Bytes(f_wavfile,pos - 8);

	 fseek(f_wavfile,40,SEEK_SET);
	 Write4Bytes(f_wavfile,pos - 44);

	 fclose(f_wavfile);
	 f_wavfile = NULL;

 } // end of CloseWavFile
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// Age voice age in years, 0 for not specified.
type Age int

// Variant after a list of candidates is produced, scored and sorted,
// "variant" is used to index that list and choose a voice.
// variant=0 takes the top voice (i.e. best match). variant=1
// takes the next voice, etc
type Variant int

// Gender voice gender.
type Gender int

const (
	// Unspecified or none.
	Unspecified Gender = iota
	// Male voice variant.
	Male
	// Female voice variant.
	Female
)

// Voice analogous to C.espeak_VOICE. New voices can be created as long as
// they're listed in "espeak --voices=<lang>".
type Voice struct {
	Name       string
	Languages  string
	Identifier string
	Gender     Gender
	Age        Age
	Variant    Variant
}

func (v *Voice) cptr() *C.espeak_VOICE {
	cv := &cVoice{
		name:       C.CString(v.Name),
		languages:  C.CString(v.Languages),
		identifier: C.CString(v.Identifier),
		gender:     C.uchar(int(v.Gender)),
		age:        C.uchar(int(v.Age)),
		variant:    C.uchar(int(v.Variant)),
	}
	return (*C.espeak_VOICE)(unsafe.Pointer(cv))
}

// Default voices.
var (
	DefaultVoice   = ENUSMale
	ENUSMale       = &Voice{Name: "english-us", Languages: "en-us", Identifier: "en-us", Gender: Male}
	ENUSFemale     = &Voice{Name: "us-mbrola-1", Languages: "en-us", Identifier: "mb/mb-us1", Gender: Female}
	ENUKMale       = &Voice{Name: "english-mb-en1", Languages: "en-uk", Identifier: "mb/mb-en1", Gender: Male}
	ESSpainMale    = &Voice{Name: "spanish", Languages: "es", Identifier: "europe/es", Gender: Male}
	ESLatinMale    = &Voice{Name: "spanish-latin-am", Languages: "es-la", Identifier: "es-la", Gender: Male}
	ESMexicanMale  = &Voice{Name: "mexican-mbrola-1", Languages: "es-mx", Identifier: "mb/mb-es1", Gender: Male}
	FRFranceMale   = &Voice{Name: "french", Languages: "fr-fr", Identifier: "fr", Gender: Male}
	FRFranceFemale = &Voice{Name: "french-mbrola-4", Languages: "fr", Identifier: "mb/mb-fr4", Gender: Female}
)

// cVoice analogous to espeak_VOICE.
type cVoice struct {
	name       *C.char
	languages  *C.char
	identifier *C.char
	gender     C.uchar
	age        C.uchar
	variant    C.uchar
}

// PunctType punctuation to announce.
type PunctType int

const (
	// PunctNone do not announce any punctuation.
	PunctNone PunctType = iota
	// PunctAll announce all punctuation signs.
	PunctAll
	// PunctSome only announce punctuation signs as defined by
	// &Parameters.PunctuationList() or set by SetPunctList.
	PunctSome
)

func (p PunctType) String() string {
	return [...]string{
		PunctNone: "Punctuation type: None",
		PunctAll:  "Punctuation type: All",
		PunctSome: "Punctuation type: Some",
	}[p]
}

// Capitals setting to announce capital letters by
type Capitals int

const (
	// CapitalNone announce no capitals.
	CapitalNone Capitals = iota
	// CapitalSoundIcon distinctive sound for capitals.
	CapitalSoundIcon
	// CapitalSpelling spells out "Capital A" for each capital.
	CapitalSpelling
	// CapitalPitchRaise uses a different pitch for capital letters.
	CapitalPitchRaise
)

func (c Capitals) String() string {
	return [...]string{
		CapitalNone:       "Capitals: None",
		CapitalSoundIcon:  "Capitals: Sound icon",
		CapitalSpelling:   "Capitals: Spelling",
		CapitalPitchRaise: "Capitals: PitchRaise",
	}[c]
}

// Parameters espeak voice parameters.
type Parameters struct {
	// Rate speaking speed in word per minute.  Values 80 to 450. Default 160.
	Rate int
	// Volume in range 0-200 or more.
	// 0=silence, 100=normal full volume, greater values may
	// produce amplitude compression or distortion. Default 100.
	Volume int
	// Pitch base pitch. Range 0-100. Default 50 (normal).
	Pitch int
	// Range pitch range, range 0-100. 0-monotone, 50=normal. Default 50 (normal).
	Range int
	// AnnouncePunctuation settings. See PunctType for details. Default All (2).
	AnnouncePunctuation PunctType
	// AnnounceCapitals settings. See Capitals for details. Default None (0).
	AnnounceCapitals Capitals
	// WordGap between words. Default 10.
	WordGap int
	// Dir directory path to save .wav files. Default os.TempDir()
	Dir       string
	punctList string
}

// PunctuationList returns the list of punctuation characters (if any).
func (p *Parameters) PunctuationList(chars string) string {
	return p.punctList
}

// SetPunctuationList sets the list of punctuation characters.
func (p *Parameters) SetPunctuationList(chars string) {
	p.punctList = chars
}

func (p *Parameters) setVoiceParams() error {
	var ee C.espeak_ERROR
	ee = C.espeak_SetParameter(C.espeakWORDGAP, C.int(p.WordGap), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakRATE, C.int(p.Rate), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakVOLUME, C.int(p.Volume), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakPITCH, C.int(p.Pitch), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakRANGE, C.int(p.Range), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakPUNCTUATION, C.int(p.AnnouncePunctuation), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakCAPITALS, C.int(p.AnnounceCapitals), C.int(0))
	if err := errFromCode(ee); err != nil {
		return err
	}
	if p.punctList != "" {
		ee = C.espeak_SetPunctuationList((*C.wchar_t)(unsafe.Pointer(C.CString(p.punctList))))
		if err := errFromCode(ee); err != nil {
			return err
		}
	}
	return nil
}

// Option parameter creatien function.
type Option func(*Parameters)

// DefaultParameters for voice modulation.
var DefaultParameters = &Parameters{
	Rate:                160,
	Volume:              100,
	Pitch:               50,
	Range:               50,
	AnnouncePunctuation: PunctAll,
	AnnounceCapitals:    CapitalNone,
	WordGap:             10,
	Dir:                 os.TempDir(),
}

// NewParameters returns *DefaultParameters modified by opts.
func NewParameters(opts ...Option) *Parameters {
	p := DefaultParameters
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithRate rate.
func WithRate(rate int) Option {
	return func(p *Parameters) {
		p.Rate = rate
	}
}

// WithVolume volume.
func WithVolume(volume int) Option {
	return func(p *Parameters) {
		p.Volume = volume
	}
}

// WithPitch pitch.
func WithPitch(pitch int) Option {
	return func(p *Parameters) {
		p.Pitch = pitch
	}
}

// WithRange rng.
func WithRange(rng int) Option {
	return func(p *Parameters) {
		p.Range = rng
	}
}

// WithAnnouncePunctuation punct.
func WithAnnouncePunctuation(punct PunctType) Option {
	return func(p *Parameters) {
		p.AnnouncePunctuation = punct
	}
}

// WithAnnounceCapitals cap.
func WithAnnounceCapitals(cap Capitals) Option {
	return func(p *Parameters) {
		p.AnnounceCapitals = cap
	}
}

// WithWordGap wg.
func WithWordGap(wg int) Option {
	return func(p *Parameters) {
		p.WordGap = wg
	}
}

// WithDir path.
func WithDir(path string) Option {
	return func(p *Parameters) {
		p.Dir = path
	}
}

// TextToSpeech reproduces text, using voice, modified by params.
// If params is nil, default parameters are used.
// If outfile is an empty string or "play", the audio is spoken to the system
// default's audio output; otherwise is appended with .wav and saved to
// params.Dir/outfile[.wav]. Returns the number of samples written to file
// if any.
func TextToSpeech(text string, voice *Voice, outfile string, params *Parameters) (uint64, error) {
	if text == "" {
		return 0, ErrEmptyText
	}
	if params == nil {
		params = NewParameters()
	}
	if voice == nil {
		voice = DefaultVoice
	}

	var (
		size             C.ulong = C.ulong(len(text))
		options          C.int   = C.espeakINITIALIZE_PHONEME_EVENTS
		position         C.uint  = 0
		positionType     C.espeak_POSITION_TYPE
		endPosition      C.uint = 0
		flags            C.uint = C.espeakCHARS_AUTO | C.espeakENDPAUSE
		output           C.espeak_AUDIO_OUTPUT
		uniqueIdentifier *C.uint
		userData         unsafe.Pointer
		ctext            *C.char
		// bufLength length in mS of sound buffers passed to the SynthCallback
		// function. Value=0 gives a default of 200mS
		bufLength C.int = 100
		// path directory which contains the espeak-data directory, NULL for
		// the default location.
		path *C.char
		ee   C.espeak_ERROR
	)

	if outfile == "" || outfile == "play" {
		output = C.AUDIO_OUTPUT_PLAYBACK
	} else {
		output = C.AUDIO_OUTPUT_SYNCHRONOUS
	}
	ctext = C.CString(text)
	defer C.free(unsafe.Pointer(ctext))

	outfile = ensureWavSuffix(outfile)
	if err := os.MkdirAll(params.Dir, 0755); err != nil {
		return 0, err
	}
	outfile = filepath.Join(params.Dir, outfile)

	C.wavefile = C.CString(outfile)
	defer C.free(unsafe.Pointer(C.wavefile))

	C.samplerate = C.espeak_Initialize(output, bufLength, path, options)
	if int(C.samplerate) == -1 {
		return 0, EErrInternal
	}
	if err := params.setVoiceParams(); err != nil {
		return 0, err
	}

	//set call back
	C.espeak_SetSynthCallback((*C.t_espeak_callback)(C.SynthCallback))

	ee = C.espeak_SetVoiceByProperties(voice.cptr())
	if err := errFromCode(ee); err != nil {
		return 0, err
	}
	ee = C.espeak_Synth(
		unsafe.Pointer(ctext),
		size,
		position,
		positionType,
		endPosition,
		flags,
		uniqueIdentifier,
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

type EspeakError struct {
	code int
	err  string
}

func (e *EspeakError) Error() string {
	return fmt.Sprintf("espeak: (%d) %s", e.code, e.err)
}

// Errors
var (
	// EErrOK espeak return for not-really-an-error.
	EErrOK = &EspeakError{0, "OK"}
	// EErrInternal espeak return for internal error
	EErrInternal = &EspeakError{-1, "Internal error"}
	// EErrBufferFull espeak buffer full error.
	EErrBufferFull = &EspeakError{1, "Buffer full"}
	// EErrNotFound espeak not found error.
	EErrNotFound = &EspeakError{2, "Not found"}
	// ErrEmptyText text is empty.
	ErrEmptyText = errors.New("text is empty")
	// ErrUnknown unknown error code.
	ErrUnknown = errors.New("unknown error code")
)

func errFromCode(code C.espeak_ERROR) error {
	switch code {
	case C.EE_OK:
		return nil
	case C.EE_INTERNAL_ERROR:
		return EErrInternal
	case C.EE_BUFFER_FULL:
		return EErrBufferFull
	case C.EE_NOT_FOUND:
		return EErrNotFound
	default:
		return ErrUnknown
	}
}
