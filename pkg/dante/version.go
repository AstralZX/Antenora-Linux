// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"strconv"
	"strings"
	"unicode"
)

// CompareVersions compares two version strings and returns:
//
//	-1 if a < b, 0 if a == b, +1 if a > b.
//
// It implements a Debian/Gentoo-style comparison that treats digit runs
// numerically and non-digit runs lexically, so "6.10" > "6.9" and
// "1.0_rc1" < "1.0". A leading "v" and trailing release markers such as
// "cachyos" are handled gracefully.
func CompareVersions(a, b string) int {
	as := versionSegments(a)
	bs := versionSegments(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		if i >= len(as) {
			// a is exhausted: a pre-release marker still pending in b means
			// b ranks lower than a release.
			if isPreRelease(bs[i]) {
				return 1
			}
			return -1
		}
		if i >= len(bs) {
			if isPreRelease(as[i]) {
				return -1
			}
			return 1
		}
		if c := compareSegment(as[i], bs[i]); c != 0 {
			return c
		}
	}
	return 0
}

// isPreRelease reports whether a segment is a pre-release marker
// (alpha/beta/pre/rc) rather than a release or arbitrary suffix.
func isPreRelease(s versionSeg) bool {
	return !s.num && releaseRank(s.str) < 4
}

type versionSeg struct {
	num    bool
	str    string
	numVal int64
}

func versionSegments(v string) []versionSeg {
	v = strings.TrimPrefix(v, "v")
	var out []versionSeg
	runes := []rune(v)
	i := 0
	for i < len(runes) {
		if runes[i] == '.' || runes[i] == '-' || runes[i] == '_' || runes[i] == '+' {
			i++
			continue
		}
		j := i
		for j < len(runes) && unicode.IsDigit(runes[j]) {
			j++
		}
		if j > i {
			numStr := string(runes[i:j])
			n, _ := strconv.ParseInt(numStr, 10, 64)
			out = append(out, versionSeg{num: true, str: numStr, numVal: n})
			i = j
			continue
		}
		j = i
		for j < len(runes) && !unicode.IsDigit(runes[j]) && runes[j] != '.' && runes[j] != '-' && runes[j] != '_' && runes[j] != '+' {
			j++
		}
		if j > i {
			out = append(out, versionSeg{num: false, str: string(runes[i:j])})
			i = j
			continue
		}
		i++
	}
	return out
}

func compareSegment(a, b versionSeg) int {
	// A numeric segment always sorts before an alpha segment (1.0 < 1.0a).
	if a.num && !b.num {
		return -1
	}
	if !a.num && b.num {
		return 1
	}
	if a.num {
		switch {
		case a.numVal < b.numVal:
			return -1
		case a.numVal > b.numVal:
			return 1
		default:
			return 0
		}
	}
	// Alpha comparison: rc/beta/alpha/pre markers rank lower than release.
	ra := releaseRank(a.str)
	rb := releaseRank(b.str)
	if ra != rb {
		if ra < rb {
			return -1
		}
		return 1
	}
	return strings.Compare(a.str, b.str)
}

// releaseRank orders pre-release keywords. Empty (release) ranks highest.
func releaseRank(s string) int {
	switch strings.ToLower(s) {
	case "alpha", "a":
		return 0
	case "beta", "b":
		return 1
	case "pre", "preview":
		return 2
	case "rc", "cr":
		return 3
	case "":
		return 5
	default:
		return 4
	}
}
