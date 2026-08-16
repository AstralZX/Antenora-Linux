// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"1.1", "1.0", 1},
		{"1.0", "1.1", -1},
		{"6.10", "6.9", 1},
		{"6.9.6", "6.9.5", 1},
		{"2.0.0", "1.9999", 1},
		{"1.0_rc1", "1.0", -1},
		{"1.0", "1.0_rc1", 1},
		{"1.0alpha", "1.0beta", -1},
		{"v2.0", "v1.9", 1},
		{"6.9.6-cachyos", "6.9.5-cachyos", 1},
		{"6.9.6-cachyos", "6.9.6-cachyos", 0},
		{"1.0.0", "1.0.0", 0},
		{"3.12.4", "3.12.10", -1},
		{"", "1.0", -1},
		{"1.0", "", 1},
		{"", "", 0},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCompareVersionsSymmetry(t *testing.T) {
	pairs := [][2]string{
		{"1.2.3", "1.2.4"},
		{"0.9", "1.0"},
		{"2.39", "2.40"},
		{"8.8.0", "8.8.0"},
	}
	for _, p := range pairs {
		ab := CompareVersions(p[0], p[1])
		ba := CompareVersions(p[1], p[0])
		if ab == 0 && ba != 0 {
			t.Fatalf("CompareVersions asymmetric for equal pair %v", p)
		}
		if ab != 0 && ab != -ba {
			t.Fatalf("CompareVersions asymmetric for %v: ab=%d ba=%d", p, ab, ba)
		}
	}
}
