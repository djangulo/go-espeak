// Copyright 2020 djangulo. All rights reserved. Use of this source code is
// governed by an MIT license that can be found in the LICENSE file.
package espeak_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/djangulo/go-espeak"
	"github.com/djangulo/go-espeak/native"
)

func BenchmarkTextToSpeech(b *testing.B) {
	tmp, err := ioutil.TempDir("", "go-espeak-benchmarks-default-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	p := espeak.NewParameters().WithDir(tmp)
	defer espeak.Terminate()
	for i := 0; i < b.N; i++ {
		espeak.TextToSpeech("Hello world!", nil, "test-espeak.wav", p)
	}
}

func BenchmarkCNativeTextToSpeech(b *testing.B) {
	tmp, err := ioutil.TempDir("", "go-espeak-benchmarks-native*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	p := espeak.NewParameters().WithDir(tmp)
	defer espeak.Terminate()
	for i := 0; i < b.N; i++ {
		native.TextToSpeech("Hello world!", nil, "test-native.wav", p)
	}
}

// func BenchmarkNativeTextToSpeech(b *testing.B) {
// 	defer espeak.Terminate()
// 	for i := 0; i < b.N; i++ {
// 		native.TextToSpeech("Hello world!", nil, "test-native.wav", nil)
// 	}
// }
