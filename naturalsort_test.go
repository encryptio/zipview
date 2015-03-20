package main

import (
	"testing"
)

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		A, B    string
		Compare int // -1 is <, 0 is =, 1 is >
	}{
		{"a", "b", -1},
		{"1", "2", -1},
		{"01", "02", -1},
		{"01", "01", 0},
		{"image_1", "image_10", -1},
		{"image_1a", "image_000001b", -1},
		{"image_1b", "image_000001a", 1},
		{"a1", "A2", -1},
		{"A1", "a2", -1},
		{"A", "a", -1},
		{"01", "1", -1},
	}

	for _, test := range tests {
		altb := naturalLess(test.A, test.B)
		blta := naturalLess(test.B, test.A)

		wantAltB := test.Compare < 0
		wantBltA := test.Compare > 0

		if altb != wantAltB || blta != wantBltA {
			t.Errorf("comparing %#v to %#v, want comparison %v, but got a<b=%v, b<a=%v",
				test.A, test.B, test.Compare, altb, blta)
		}
	}
}
