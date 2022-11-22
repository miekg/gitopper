package main

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSliceFlag(t *testing.T) {
	for _, test := range []struct {
		in  string
		out []string
	}{
		{
			in:  "",
			out: []string{""},
		},
		{
			in:  "hi",
			out: []string{"hi"},
		},
		{
			in:  "hi,mom",
			out: []string{"hi", "mom"},
		},
	} {
		var out []string
		f := sliceFlag{&out}
		if got, want := f.Set(test.in), error(nil); !errors.Is(got, want) {
			t.Fatalf("f.Set(%q) = %v, want %v", test.in, got, want)
		}
		if diff := cmp.Diff(test.out, out); diff != "" {
			t.Errorf("after f.Set(%q), out = %v, want %v\n\ndiff:\n%v", test.in, out, test.out, diff)
		}
		if got, want := f.String(), test.in; got != want {
			t.Errorf("after f.Set(%q), f.String() = %q, want %q", test.in, got, want)
		}
	}
}
