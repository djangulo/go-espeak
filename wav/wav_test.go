// Copyright 2020 djangulo. All rights reserved. Use of this source code is
// governed by an MIT license that can be found in the LICENSE file.
package wav

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkInt32ToBytes(b *testing.B) {
	w := newWavHeader()
	for i := 0; i < b.N; i++ {
		w.littleEndianInt32ToBytes(40, 1<<16-1)
	}
}

func BenchmarkInt32ToBytesBinary(b *testing.B) {
	w := newWavHeader()
	for i := 0; i < b.N; i++ {
		w.littleEndianInt32ToBytesBinary(40, 1<<16-1)
	}
}

func TestWriter_WriteSamples(t *testing.T) {
	tmp, err := ioutil.TempDir("", "go-espeak-wav-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	fh, err := os.Create(filepath.Join(tmp, "test-wav.wav"))
	if err != nil {
		t.Fatal(err)
	}
	w := NewWriter(fh, 44100)
	in := []int16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	w.WriteSamples(in)
	fh.Close()

	fh, err = os.Open(filepath.Join(tmp, "test-wav.wav"))
	if err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 44, 44)
	io.ReadFull(fh, got)
	head := newWavHeader()
	for i := 0; i < 4; i++ {
		if head[i] != got[i] {
			t.Errorf("expected byte %d to be %v, instead got %v", i, head[i], got[i])
		}
	}
	// data size in bytes is len(in)*2
	if got[40] != byte(len(in)*2) {
		t.Errorf("expected byte 40 to be %v, instead got %v", len(in), got[40])
	}
}
