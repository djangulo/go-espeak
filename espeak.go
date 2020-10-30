//Package espeak implements C bindings for the Espeak voice synthesizer.
// It also provides Go wrappers around espeak's api that allow for
// creation of custom text synthesis functions.
package espeak

/*
#cgo CFLAGS: -I/usr/include/espeak
#cgo LDFLAGS: -l portaudio -l espeak
#include <stdio.h>
#include <string.h>
#include <malloc.h>
#include <speak_lib.h>
void* user_data;
unsigned int samplestotal = 0;
int samplerate;
char *wavefile=NULL;
FILE *f_wavfile = NULL;
int OpenWavFile(char *path, int rate);
void CloseWavFile();
int SynthCallback(short *wav, int numsamples, espeak_EVENT *events);
const espeak_VOICE **ListVoices(espeak_VOICE *voice_spec, unsigned int *count);

// ListVoices calls espeak_ListVoices, returns a null-terminated array of
// matching *espeak_VOICE objects and populates count with the length of said
// array.
const espeak_VOICE **ListVoices(espeak_VOICE *voice_spec, unsigned int *count)
{
    const espeak_VOICE **out;
    out = espeak_ListVoices(voice_spec);
    unsigned int i = 0;
    while (out[i] != NULL)
    {
        i++;
    }
    *count = i;
    return out;
}
int SynthCallback(short *wav, int numsamples, espeak_EVENT *events)
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

// WordToString: Convert a phoneme mnemonic word into a string
const char *WordToString(unsigned int word)
{
    int ix;
    static char buf[5];
    for (ix = 0; ix < 3; ix++)
    {
        buf[ix] = word >> (ix * 8);
    }
    buf[4] = 0;
    return (buf);
}
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	var length C.uint = 0
	// out is Ctype const espeak_VOICE **
	out := C.ListVoices(voiceSpec, &length)
	defer C.free(unsafe.Pointer(out))
	cVoices := (*[1 << 28]*C.espeak_VOICE)(unsafe.Pointer(out))[:length:length]
	voices = make([]*Voice, 0)
	for _, cv := range cVoices {
		voices = append(voices, voiceFromCptr(unsafe.Pointer(cv)))
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
)

// Init wrapper around espeak_Initialize.
//   - output AudioOutput type.
//   - bufferLength length in mS of sound buffers passed to the SynthCallback
//     function. If 0 gives a default of 200mS. Only used for
//     output==Retrieval and output == Synchronous.
//   - path: the directory which contains the espeak-data directory.
//   - options: InitOption to use.
func Init(
	output AudioOutput,
	bufferLength int,
	path string,
	options InitOption,
) (int, error) {
	if bufferLength == 0 {
		bufferLength = 200
	}

	if initialized {
		return 0, ErrAlreadyInitialized
	}
	if options&UseMbrola == UseMbrola {
		useMbrola = true
	}
	var cPath *C.char
	if path != "" {
		cPath = C.CString(path)
		defer C.free(unsafe.Pointer(cPath))
	}
	var sr C.int
	sr = C.espeak_Initialize(
		output.toC(),
		C.int(bufferLength),
		cPath,
		C.int(options),
	)
	if int(sr) == -1 {
		return 0, EErrInternal
	}
	C.samplerate = sr
	initialized = true
	return int(sr), nil
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
	uniqueIdent *C.uint,
	userData unsafe.Pointer,
) error {
	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))

	ee := C.espeak_Synth(
		unsafe.Pointer(ctext),
		C.ulong(len(text)),
		C.uint(startPos),
		posType.toC(),
		C.uint(endPos),
		C.uint(flags),
		uniqueIdent,
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

	var (
		uniqueIdentifier *C.uint
		userData         unsafe.Pointer
		// bufLength length in mS of sound buffers passed to the SynthCallback
		// function. Value=0 gives a default of 200mS
		bufLength int = 200
		output    AudioOutput
	)

	if outfile == "" || outfile == "play" {
		output = Playback
	} else {
		output = Synchronous
	}

	outfile = ensureWavSuffix(outfile)
	if err := os.MkdirAll(params.Dir, 0755); err != nil {
		return 0, err
	}
	outfile = filepath.Join(params.Dir, outfile)

	C.wavefile = C.CString(outfile)
	defer C.free(unsafe.Pointer(C.wavefile))

	_, err := Init(output, bufLength, "", PhonemeEvents)
	// if the error is of type ErrAllreadyInitialized, continue
	if err != nil && !errors.Is(err, ErrAlreadyInitialized) {
		return 0, err
	}

	if err := params.SetVoiceParams(); err != nil {
		return 0, err
	}

	//set call back
	SetSynthCallback(C.SynthCallback)
	if err := SetVoiceByName(voice.Name); err != nil {
		return 0, err
	}

	if err := Synth(text, CharsAuto|EndPause, 0, 0, Character, uniqueIdentifier, userData); err != nil {
		return 0, err
	}
	if err := Synchronize(); err != nil {
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
