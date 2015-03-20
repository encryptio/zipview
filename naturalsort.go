package main

import (
	"archive/zip"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type sortableZIPList []*zip.File

func (s sortableZIPList) Len() int      { return len(s) }
func (s sortableZIPList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableZIPList) Less(i, j int) bool {
	return naturalLess(s[i].Name, s[j].Name)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func naturalLess(oa, ob string) bool {
	a := oa
	b := ob

	for len(a) > 0 && len(b) > 0 {
		if isDigit(a[0]) && isDigit(b[0]) {
			ai := 1
			for ai < len(a) && isDigit(a[ai]) {
				ai++
			}
			bi := 1
			for bi < len(b) && isDigit(b[bi]) {
				bi++
			}

			an, _ := strconv.ParseUint(a[:ai], 10, 64)
			bn, _ := strconv.ParseUint(b[:bi], 10, 64)
			a = a[ai:]
			b = b[bi:]

			if an != bn {
				return an < bn
			}
		} else {
			ra, as := utf8.DecodeRuneInString(a)
			a = a[as:]
			rb, bs := utf8.DecodeRuneInString(b)
			b = b[bs:]

			ra = unicode.ToLower(ra)
			rb = unicode.ToLower(rb)

			if ra != rb {
				return ra < rb
			}
		}
	}

	// shorter compares lesser
	if len(a) > 0 {
		return true
	}
	if len(b) > 0 {
		return false
	}

	// fall back to a case-sensitive, naive comparison
	return oa < ob
}
