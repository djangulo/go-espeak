// Copyright 2020 djangulo. All rights reserved.
// Use of this source code is governed by a GPLv3
// license that can be found in the LICENSE file.
package espeak

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

func TestTextToSpeech(t *testing.T) {
	tmp, err := ioutil.TempDir("", "go-espeak-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	p := NewParameters(WithDir(tmp))
	t.Run("success", func(t *testing.T) {
		samples, err := TextToSpeech("test speech", nil, "test", p)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if samples == 0 {
			t.Errorf("0 samples written")
		}
	})
	t.Run("errors", func(t *testing.T) {
		for _, tt := range []struct {
			name   string
			text   string
			params *Parameters
			voice  *Voice
			want   error
		}{
			{"empty text", "", nil, nil, ErrEmptyText},
		} {
			t.Run(tt.name, func(t *testing.T) {
				s, err := TextToSpeech(tt.text, nil, "test", p)
				if s != 0 {
					t.Errorf("expected return samples 0 got %d", s)
				}
				if !errors.Is(err, tt.want) {

				}
				if err == nil {
					t.Error("expected an error but didn't get one")
				}
			})
		}
	})

}
