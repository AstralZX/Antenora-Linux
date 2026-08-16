// Copyright (C) 2026 Antenora Linux contributors
// SPDX-License-Identifier: AGPL-3.0-or-later

package dante

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// LoadMirrors reads the mirror list from the configured mirror file. Each
// non-comment line is a base URL. Returns nil if the file is absent.
func (d *Dante) LoadMirrors() ([]string, error) {
	f, err := os.Open(d.Config.MirrorFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.TrimRight(line, "/"))
	}
	return out, sc.Err()
}

// Mirrors returns the ordered list of binary mirrors: configured mirrors
// first, then the default BinaryMirror as the final fallback.
func (d *Dante) Mirrors() []string {
	m, _ := d.LoadMirrors()
	if len(m) == 0 {
		return []string{strings.TrimRight(d.Config.BinaryMirror, "/")}
	}
	return append(m, strings.TrimRight(d.Config.BinaryMirror, "/"))
}

// TestMirrors probes each mirror and prints its latency and status, fastest
// first. Use this to write an ordered mirror list.
func (d *Dante) TestMirrors() error {
	mirrors := d.Mirrors()
	type result struct {
		url string
		ms  time.Duration
		ok  bool
	}
	results := make([]result, 0, len(mirrors))
	client := &http.Client{Timeout: 10 * time.Second}
	for _, m := range mirrors {
		start := time.Now()
		resp, err := client.Head(m + "/")
		ms := time.Since(start)
		ok := err == nil && resp != nil && resp.StatusCode < 500
		if resp != nil {
			resp.Body.Close()
		}
		results = append(results, result{m, ms, ok})
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].ms < results[j].ms })
	fmt.Printf("%-6s %10s  %s\n", "STATUS", "LATENCY", "MIRROR")
	for _, r := range results {
		status := "OK"
		if !r.ok {
			status = "DOWN"
		}
		fmt.Printf("%-6s %10s  %s\n", status, r.ms.Round(time.Millisecond), r.url)
	}
	return nil
}
