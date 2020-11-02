// Copyright 2020 djangulo. All rights reserved. Use of this source code is
// governed by an MIT license that can be found in the LICENSE file.

//Package espeak implements C bindings for the Espeak voice synthesizer.
// It also provides Go wrappers around espeak's api that allow for
// creation of custom text synthesis functions.
package espeak

/*
#cgo CFLAGS: -I/usr/include/espeak
#cgo LDFLAGS: -lportaudio -lespeak
#include <stdio.h>
#include <string.h>
#include <malloc.h>
#include <speak_lib.h>

static inline void *eventUserData(espeak_EVENT *event)  {
	if (event != NULL)
		if (event->user_data != NULL)
			return event->user_data;

	return NULL;
}

extern int processSamples(short *wav, int numsamples, espeak_EVENT *events);
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/djangulo/go-espeak/wav"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Age voice age in years, 0 for not specified.
type Age int

// Variant after a list of candidates is produced, scored and sorted,
// "variant" is used to index that list and choose a voice.
// variant=0 takes the top voice (i.e. best match). variant=1
// takes the next voice, etc
type Variant int

// Gender voice gender.
type Gender int

// String implements the stringer interface.
func (g Gender) String() string {
	switch g {
	case Male:
		return "M"
	case Female:
		return "F"
	default:
		return "-"
	}
}

const (
	// Unspecified or none.
	Unspecified Gender = iota
	// Male voice variant.
	Male
	// Female voice variant.
	Female
)

// UnmarshalJSON implements the JSON.Unmarshaler interface.
func (g *Gender) UnmarshalJSON(data []byte) (err error) {
	switch v := data; {
	case bytes.Equal(v, []byte(`"M"`)) || bytes.Equal(v, []byte(`"m"`)):
		*g = Male
	case bytes.Equal(v, []byte(`"F"`)) || bytes.Equal(v, []byte(`"f"`)):
		*g = Female
	default:
		*g = Unspecified
	}
	return nil
}

// MarshalJSON marshals the Gender into a string.
func (g Gender) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.String())
}

// Voice analogous to C.espeak_VOICE. New voices can be created as long as
// they're listed in "espeak --voices=<lang>".
type Voice struct {
	Name       string `json:"name,omitempty"`
	Languages  string `json:"languages,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Gender     Gender `json:"gender,omitempty"`
	Age        Age
	Variant    Variant
}

// Default voices.
var (
	DefaultVoice = ENUSMale
	ENUSMale     = &Voice{Name: "english-us", Languages: "en-us", Identifier: "en-us", Gender: Male}
	ESSpainMale  = &Voice{Name: "spanish", Languages: "es", Identifier: "europe/es", Gender: Male}
	ESLatinMale  = &Voice{Name: "spanish-latin-am", Languages: "es-la", Identifier: "es-la", Gender: Male}
	FRFranceMale = &Voice{Name: "french", Languages: "fr-fr", Identifier: "fr", Gender: Male}
)

func (v *Voice) String() string {
	return fmt.Sprintf("%s:%s(%s)[%s]", v.Languages, v.Name, v.Identifier, v.Gender)
}

// VoiceFromSpec returns a random Voice from the group of voices that matches
// spec. Is spec is nil, returns a random voice.
func VoiceFromSpec(spec *Voice) (*Voice, error) {
	candidates, err := ListVoices(spec)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, EErrNotFound
	}
	res := make([]*Voice, 0)
	if !useMbrola {
		for _, c := range candidates {
			if !strings.HasPrefix(c.Identifier, "mb") {
				res = append(res, c)
			}
		}
	} else {
		res = candidates
	}

	rand.Seed(time.Now().Unix())
	i := rand.Intn(len(res))
	return res[i], nil
}

// ListVoices reads the voice files from espeak-data/voices and returns them
// in a []*Voice object. If spec is nil, all available voices are listed.
// If spec is given, then only the voices which are compatible with the spec
// are listed, and they are listed in preference order.
// Init must have been called.
func ListVoices(spec *Voice) (voices []*Voice, err error) {
	if !initialized {
		return nil, ErrNotInitialized
	}
	var voiceSpec *C.espeak_VOICE
	if spec != nil {
		voiceSpec = spec.cptr()
	}
	// out is Ctype const espeak_VOICE ** (pointer to array)
	out := C.espeak_ListVoices(voiceSpec)
	defer C.free(unsafe.Pointer(out))
	// slice-ification of a C.espeak_VOICE ** into a slice
	//     (*C.espeak_VOICE)(unsafe.Pointer(out)) gets the actual array
	//     length is fixed at a 1000 as we don't know how many return
	//     espeak_ListVoices returns a NULL terminated array
	cVoices := (*[1 << 28]*C.espeak_VOICE)(
		unsafe.Pointer(
			(*C.espeak_VOICE)(unsafe.Pointer(out))))[:1000:1000]
	defer func() {
		cVoices = nil
		out = nil
	}() // garbage collect
	voices = make([]*Voice, 0)
	for _, cv := range cVoices {
		if cv == nil {
			break
		}
		if useMbrola {
			voices = append(voices, voiceFromCptr(unsafe.Pointer(cv)))
		} else {
			if !strings.HasPrefix(C.GoString(cv.identifier), "mb") {
				voices = append(voices, voiceFromCptr(unsafe.Pointer(cv)))
			}
		}
	}

	return voices, nil
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

func voiceFromCptr(ptr unsafe.Pointer) *Voice {
	return ((*cVoice)(ptr)).goVoice()
}

// cVoice analogous to espeak_VOICE.
type cVoice struct {
	name       *C.char
	languages  *C.char
	identifier *C.char
	gender     C.uchar
	age        C.uchar
	variant    C.uchar
}

func (cv *cVoice) goVoice() *Voice {
	return &Voice{
		Name:       C.GoString(cv.name),
		Languages:  C.GoString(cv.languages),
		Identifier: C.GoString(cv.identifier),
		Gender:     Gender(cv.gender),
		Age:        Age(cv.age),
		Variant:    Variant(cv.variant),
	}
}

// PositionType determines whether "position" is a number of characters,
// words, or sentences.
type PositionType uint8

func (pt PositionType) toC() C.espeak_POSITION_TYPE {
	switch pt {
	case Word:
		return C.POS_WORD
	case Sentence:
		return C.POS_SENTENCE
	default: //POS_CHARACTER
		return C.POS_CHARACTER
	}
}

const (
	// Character position type.
	Character PositionType = iota + 1
	// Word position type.
	Word
	// Sentence position type.
	Sentence
)

// PunctType punctuation to announce.
type PunctType int

func (p PunctType) toC() C.int {
	switch p {
	case PunctAll:
		return C.espeakPUNCT_ALL
	case PunctSome:
		return C.espeakPUNCT_SOME
	default: // None
		return C.espeakPUNCT_NONE
	}
}

const (
	// PunctNone do not announce any punctuation.
	PunctNone PunctType = 0
	// PunctAll announce all punctuation signs.
	PunctAll PunctType = 1
	// PunctSome only announce punctuation signs as defined by
	// &Parameters.PunctuationList() or set by SetPunctList.
	PunctSome PunctType = 2
)

func (p PunctType) String() string {
	return [...]string{
		PunctNone: "Punctuation type: None",
		PunctAll:  "Punctuation type: All",
		PunctSome: "Punctuation type: Some",
	}[p]
}

// Capitals setting to announce capital letters by.
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
	// Rate speaking speed in word per minute.  Values 80 to 450. Default 175.
	Rate int
	// Volume in range 0-200 or more.
	// 0=silence, 100=normal full volume, greater values may
	// produce amplitude compression or distortion. Default 100.
	Volume int
	// Pitch base pitch. Range 0-100. Default 50 (normal).
	Pitch int
	// Range pitch range, range 0-100. 0-monotone, 50=normal. Default 50 (normal).
	Range int
	// AnnouncePunctuation settings. See PunctType for details. Default None (0).
	AnnouncePunctuation PunctType
	// AnnounceCapitals settings. See Capitals for details. Default None (0).
	AnnounceCapitals Capitals
	// WordGap pause between words, units of 10mS (at the default speed).
	WordGap int
	// Dir directory path to save .wav files. Default os.TempDir()
	Dir       string
	punctList string
}

// PunctuationList returns the list of punctuation characters (if any).
func (p *Parameters) PunctuationList() string {
	return p.punctList
}

// SetPunctuationList sets the list of punctuation characters.
func (p *Parameters) SetPunctuationList(chars string) {
	p.punctList = chars
}

// SetVoiceParams calls espeak_SetParameter for each of the *Parameters
// fields.
func (p *Parameters) SetVoiceParams() error {
	var ee C.espeak_ERROR
	ee = C.espeak_SetParameter(C.espeakRATE, C.int(p.Rate), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakVOLUME, C.int(p.Volume), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakPITCH, C.int(p.Pitch), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakRANGE, C.int(p.Range), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakPUNCTUATION, C.int(2), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakCAPITALS, C.int(p.AnnounceCapitals), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	ee = C.espeak_SetParameter(C.espeakWORDGAP, C.int(p.WordGap), C.int(0))
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	if p.punctList != "" {
		ee = C.espeak_SetPunctuationList((*C.wchar_t)(unsafe.Pointer(C.CString(p.punctList))))
		if err := ErrFromCode(ee); err != nil {
			return err
		}
	}
	return nil
}

// Option parameter creatien function.
type Option func(*Parameters)

// DefaultParameters for voice modulation.
var DefaultParameters = &Parameters{
	Rate:                175,
	Volume:              100,
	Pitch:               50,
	Range:               50,
	AnnouncePunctuation: PunctNone,
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

// WithRate rate.
func (p *Parameters) WithRate(rate int) *Parameters {
	p.Rate = rate
	return p
}

// WithVolume volume.
func (p *Parameters) WithVolume(volume int) *Parameters {
	p.Volume = volume
	return p
}

// WithPitch pitch.
func (p *Parameters) WithPitch(pitch int) *Parameters {
	p.Pitch = pitch
	return p
}

// WithRange rng.
func (p *Parameters) WithRange(rng int) *Parameters {
	p.Range = rng
	return p
}

// WithAnnouncePunctuation punct.
func (p *Parameters) WithAnnouncePunctuation(punct PunctType) *Parameters {
	p.AnnouncePunctuation = punct
	return p
}

// WithAnnounceCapitals cap.
func (p *Parameters) WithAnnounceCapitals(cap Capitals) *Parameters {
	p.AnnounceCapitals = cap
	return p
}

// WithWordGap wg.
func (p *Parameters) WithWordGap(wg int) *Parameters {
	p.WordGap = wg
	return p
}

// WithDir path.
func (p *Parameters) WithDir(path string) *Parameters {
	p.Dir = path
	return p
}

// InitOption initialization options. Beware only PhonemeEvents and PhonemeIPA
// are the only ones that belong to espeak.
type InitOption uint8

const (
	// PhonemeEvents allow espeakEVENT_PHONEME events.
	PhonemeEvents InitOption = 1 << iota
	// PhonemeIPA espeak events give IPA phoneme names, not eSpeak phoneme names.
	PhonemeIPA
	// UseMbrola allow usage of mbrola voices, excluded by default. This is not an
	// espeak option.
	UseMbrola
)

// AudioOutput type.
type AudioOutput uint8

const (
	// Playback plays the audio data, supplies events to the calling program.
	Playback AudioOutput = iota + 1
	// Retrieval supplies audio data and events to the calling program.
	Retrieval
	// Synchronous as Retrieval but doesn't return until synthesis is completed.
	Synchronous
	// SynchPlayback synchronous playback.
	SynchPlayback
)

func (a AudioOutput) toC() C.espeak_AUDIO_OUTPUT {
	switch a {
	case Retrieval:
		return C.AUDIO_OUTPUT_RETRIEVAL
	case Synchronous:
		return C.AUDIO_OUTPUT_SYNCHRONOUS
	case SynchPlayback:
		return C.AUDIO_OUTPUT_SYNCH_PLAYBACK
	default: // playback
		return C.AUDIO_OUTPUT_PLAYBACK
	}
}

var (
	initialized bool
	useMbrola   bool
	sampleRate  int32
)

// Init wrapper around espeak_Initialize. Returns a uintptr id which the
// address of the data block (T: *[]int16) acted on, and the sample rate
// used.
//   - output AudioOutput type.
//   - bufferLength length in mS of sound buffers passed to the SynthCallback
//     function. If 0 gives a default of 200mS. Only used for
//     output==Retrieval and output == Synchronous.
//   - path: the directory which contains the espeak-data directory.
//   - options: InitOption to use.
func Init(
	output AudioOutput,
	bufferLength int,
	path *string,
	options InitOption,
) (uintptr, int32, error) {
	if bufferLength == 0 {
		bufferLength = 200
	}

	if options&UseMbrola == UseMbrola {
		useMbrola = true
	}
	var cPath *C.char
	if path != nil {
		cPath = C.CString(*path)
		defer C.free(unsafe.Pointer(cPath))
	}
	sr := C.espeak_Initialize(
		output.toC(),
		C.int(bufferLength),
		cPath,
		C.int(options),
	)
	if int(sr) == -1 {
		return 0, 0, EErrInternal
	}
	id, _, err := registry.newData()
	if err != nil {
		return 0, 0, err
	}
	sampleRate = int32(sr)
	initialized = true
	return id, sampleRate, nil
}

// SetSynthCallback to the unsafe.Pointer passed. The underlying C object
// has to be a a function of signature
//    int (t_espeak_callback)(short*, int, espeak_EVENT*)
func SetSynthCallback(ptr unsafe.Pointer) {
	C.espeak_SetSynthCallback((*C.t_espeak_callback)(ptr))
}

// Terminate closes the espeak connection. It's up to the caller to call this
// and terminate the function.
func Terminate() error {
	ee := C.espeak_Terminate()
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	return nil
}

// SetVoiceByName wrapper around espeak_SetVoiceByName.
func SetVoiceByName(name string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	ee := C.espeak_SetVoiceByName(cName)
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	return nil
}

// SetVoiceByProps wrapper around espeak_SetVoiceByProperties.
// An *Voice is used to pass criteria to select a voice.
func SetVoiceByProps(v *Voice) error {
	ee := C.espeak_SetVoiceByProperties(v.cptr())
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	return nil
}

// FlagType one-to-one mapping to the espeak flags.
type FlagType uint16

const (
	// CharsAuto 8 bit or UTF8  (this is the default).
	CharsAuto FlagType = iota
	// CharsUTF8 utf-8 encoding.
	CharsUTF8
	// Chars8Bit the 8 bit ISO-8859 character set for the particular language.
	Chars8Bit
	// CharsWChar Wide characters (wchar_t).
	CharsWChar
	// Chars16Bit 16 bit characters.
	Chars16Bit
	// SSML Elements within < > are treated as SSML elements, or if not
	// recognised are ignored.
	SSML FlagType = 0x10
	// Phonemes Text within [[ ]] is treated as phonemes codes (in espeak's
	// Hirshenbaum encoding).
	Phonemes FlagType = 0x100
	// EndPause if set then a sentence pause is added at the end of the text.
	// If not set then this pause is suppressed.
	EndPause FlagType = 0x1000
)

// Synth wrapper around espeak_Synth.
//   - text: text to synthezise.
//   - flags: flag values to pass.
//   - startPos, endPos: start and end position in the text where speaking
//     starts and ends. If endPos is zero indicates no end position.
//   - posType: PositionType to use.
//   - uniqueIdent: This must be either NULL, or point to an integer variable
//     to which eSpeak writes a message identifier number.
//     eSpeak includes this number in espeak_EVENT messages which are the
//     result of this call of espeak_Synth().
//   - userData: a pointer (or NULL) which will be passed to the callback
//     function in espeak_EVENT messages.
func Synth(
	text string,
	flags FlagType,
	startPos, endPos uint32,
	posType PositionType,
	uniqueIdent *uint64,
	userData unsafe.Pointer,
) error {
	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))

	var uid *C.uint
	if uniqueIdent != nil {
		*uid = C.uint(*uniqueIdent)
	}

	ee := C.espeak_Synth(
		unsafe.Pointer(ctext),
		C.ulong(len(text)),
		C.uint(startPos),
		posType.toC(),
		C.uint(endPos),
		C.uint(flags),
		uid,
		userData)
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	return nil
}

// Synchronize wrapper around espeak_Synchronize.
func Synchronize() error {
	ee := C.espeak_Synchronize()
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	return nil
}

// Cancel wrapper around espeak_Cancel. Stop immediately synthesis and audio
// output of the current text. When this function returns, the audio output is
// fully stopped and the synthesizer is ready to synthesize a new message.
func Cancel() error {
	ee := C.espeak_Cancel()
	if err := ErrFromCode(ee); err != nil {
		return err
	}
	return nil
}

// IsPlaying returns whether audio is being played.
func IsPlaying() bool {
	return C.espeak_IsPlaying() == 1
}

// TextToSpeech reproduces text, using voice, modified by params.
// If params is nil, default parameters are used.
// If outfile is an empty string or "play", the audio is spoken to the system
// default's audio output; otherwise is appended with .wav and saved to
// params.Dir/outfile[.wav]. Returns the number of samples written to file,
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
	// if outfile is "play" or empty, play the audio
	// this path is simple, as pretty much only the voice is set
	if outfile == "" || outfile == "play" {
		_, _, err := Init(Playback, -1, nil, PhonemeEvents)
		// if the error is of type ErrAllreadyInitialized, continue
		if err != nil && !errors.Is(err, ErrAlreadyInitialized) {
			return 0, err
		}
		if err := params.SetVoiceParams(); err != nil {
			return 0, err
		}

		if err := SetVoiceByName(voice.Name); err != nil {
			return 0, err
		}
		var uid *uint64
		if err := Synth(
			text,
			CharsAuto|EndPause,
			0,
			0,
			Character,
			uid,
			unsafe.Pointer(nil)); err != nil {
			return 0, err
		}
		if err := Synchronize(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	// outputting to wav

	// Gensamples calls init
	data, err := GenSamples(text, voice, params)
	if err != nil {
		return 0, err
	}

	outfile = ensureWavSuffix(outfile)
	if err := os.MkdirAll(params.Dir, 0755); err != nil {
		return 0, err
	}
	outfile = filepath.Join(params.Dir, outfile)
	fh, _ := os.Create(outfile)
	defer fh.Close()

	w := wav.NewWriter(fh, sampleRate)
	written, err := w.WriteSamples(data)
	if err != nil {
		return 0, nil
	}
	return written, nil
}

// GenSamples generates a []int16 sample slice containing the data of text,
// using voice, modified by params. If params is nil, default parameters are
// used.
func GenSamples(text string, voice *Voice, params *Parameters) ([]int16, error) {
	if text == "" {
		return nil, ErrEmptyText
	}
	if params == nil {
		params = NewParameters()
	}
	if voice == nil {
		voice = DefaultVoice
	}

	id, _, err := Init(Synchronous, 200, nil, PhonemeEvents)
	// if the error is of type ErrAllreadyInitialized, continue
	if err != nil && !errors.Is(err, ErrAlreadyInitialized) {
		return nil, err
	}

	var (
		uniqueIdentifier *uint64
		userData         = unsafe.Pointer(&id)
	)
	if err := params.SetVoiceParams(); err != nil {
		return nil, err
	}

	//set call back
	SetSynthCallback(C.processSamples)
	if err := SetVoiceByName(voice.Name); err != nil {
		return nil, err
	}

	if err := Synth(
		text,
		CharsAuto|EndPause,
		0,
		0,
		Character,
		uniqueIdentifier,
		userData); err != nil {
		return nil, err
	}
	if err := Synchronize(); err != nil {
		return nil, err
	}
	data := registry.getData(id)
	defer registry.removeData(id)

	return *data, nil
}

// SampleRate return the produced sample rate.
func SampleRate() int32 {
	return sampleRate
}

//export processSamples
func processSamples(wav *C.short, numsamples C.int, events *C.espeak_EVENT) C.int {
	if wav == nil {
		return 1
	}
	id := (*uintptr)(unsafe.Pointer(C.eventUserData(events)))
	data := registry.getData(*id)
	length := int(numsamples)
	*data = append(
		*data,
		(*[1 << 28]int16)(unsafe.Pointer(wav))[:length:length]...,
	)
	return 0
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

// LibError analog to espeak_ERROR.
type LibError struct {
	code int
	err  string
}

func (e *LibError) Error() string {
	return fmt.Sprintf("espeak: (%d) %s", e.code, e.err)
}

// Errors
var (
	// EErrOK espeak return for not-really-an-error.
	EErrOK = &LibError{0, "OK"}
	// EErrInternal espeak return for internal error
	EErrInternal = &LibError{-1, "Internal error"}
	// EErrBufferFull espeak buffer full error.
	EErrBufferFull = &LibError{1, "Buffer full"}
	// EErrNotFound espeak not found error.
	EErrNotFound = &LibError{2, "Not found"}
	// ErrEmptyText text is empty.
	ErrEmptyText = errors.New("text is empty")
	// ErrUnknown unknown error code.
	ErrUnknown = errors.New("unknown error code")
	// ErrAlreadyInitialized espeak already initialized.
	ErrAlreadyInitialized = errors.New("espeak already initialized")
	// ErrNotInitialized espeak not initialized (call Init).
	ErrNotInitialized = errors.New("espeak not initialized (call Init)")
)

// ErrFromCode get a Go error from an espeak_ERROR.
func ErrFromCode(code C.espeak_ERROR) error {
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

// registry keys

/*
wav [ ][ ][ ][ ]
^    ^
|    |
|    |- mem+1
|-mem spot
*/

var registry = &cache{
	samples: make([]*[]int16, 0),
}

type cache struct {
	sync.Mutex
	samples []*[]int16
}

func (c *cache) newData() (uintptr, *[]int16, error) {
	c.Lock()
	defer c.Unlock()

	d := make([]int16, 0)
	c.samples = append(c.samples, &d)
	return uintptr(unsafe.Pointer(&d)), &d, nil
}

func (c *cache) removeData(id uintptr) error {
	c.Lock()
	defer c.Unlock()

	for i, data := range c.samples {
		if uintptr(unsafe.Pointer(data)) == id {
			c.samples[i] = c.samples[len(c.samples)-1]
			c.samples[len(c.samples)-1] = nil
			c.samples = c.samples[:len(c.samples)-1]
			return nil
		}
	}

	return fmt.Errorf("id not found: %v", id)
}

// getData returns a copy of the data (as it may be deleted later)
func (c *cache) exportData(id uintptr) []int16 {
	c.Lock()
	defer c.Unlock()

	for _, data := range c.samples {
		if uintptr(unsafe.Pointer(data)) == id {
			cp := make([]int16, len(*data), cap(*data))
			copy(cp, *data)
			return cp
		}
	}

	return nil
}

// getData returns the pointer to the data.
func (c *cache) getData(id uintptr) *[]int16 {
	c.Lock()
	defer c.Unlock()

	for _, data := range c.samples {
		if uintptr(unsafe.Pointer(data)) == id {
			return data
		}
	}

	return nil
}
