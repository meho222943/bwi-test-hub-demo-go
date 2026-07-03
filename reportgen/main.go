// reportgen reads `go test -json` output on stdin and writes
// report/junit.xml + report/index.html. All-Go, standard library only,
// so the whole project runs via DIRECT_PROCESS with just the Go toolchain.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"sort"
	"strings"
)

type event struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
}

func main() {
	// action per test: "pass" | "fail" | "skip"
	results := map[string]string{}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var e event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.Test == "" {
			continue
		}
		switch e.Action {
		case "pass", "fail", "skip":
			results[e.Test] = e.Action
		}
	}

	names := make([]string, 0, len(results))
	for n := range results {
		names = append(names, n)
	}
	sort.Strings(names)

	total, failed := len(names), 0
	for _, n := range names {
		if results[n] == "fail" {
			failed++
		}
	}
	passed := total - failed

	if err := os.MkdirAll("report", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// JUnit XML
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<testsuite name="go-test" tests="%d" failures="%d">`+"\n", total, failed)
	for _, n := range names {
		if results[n] == "fail" {
			fmt.Fprintf(&b, "  <testcase classname=\"go-test\" name=\"%s\">\n", html.EscapeString(n))
			b.WriteString("    <failure message=\"test failed\"/>\n")
			b.WriteString("  </testcase>\n")
		} else {
			fmt.Fprintf(&b, "  <testcase classname=\"go-test\" name=\"%s\"/>\n", html.EscapeString(n))
		}
	}
	b.WriteString("</testsuite>\n")
	os.WriteFile("report/junit.xml", []byte(b.String()), 0o644)

	// HTML
	status, color := "PASSED", "#16a34a"
	if failed > 0 {
		status, color = "FAILED", "#dc2626"
	}
	var rows strings.Builder
	for _, n := range names {
		cls, label := "ok", "passed"
		if results[n] == "fail" {
			cls, label = "no", "failed"
		}
		fmt.Fprintf(&rows, "<tr><td>%s</td><td class=\"%s\">%s</td></tr>\n",
			html.EscapeString(n), cls, label)
	}
	page := fmt.Sprintf(`<!doctype html><html lang="en"><head><meta charset="utf-8">
<title>go test report</title>
<style>body{font-family:system-ui,sans-serif;margin:2rem;background:#0f172a;color:#e2e8f0}
.badge{display:inline-block;padding:.2rem .8rem;border-radius:999px;color:#fff;font-weight:700}
table{border-collapse:collapse;margin-top:1rem;width:100%%}td,th{padding:.5rem .8rem;border-bottom:1px solid #334155;text-align:left}
.ok{color:#4ade80}.no{color:#f87171}</style></head><body>
<h1>go test</h1>
<p>Status: <span class="badge" style="background:%s">%s</span> &middot; %d/%d passed</p>
<table><tr><th>Test</th><th>Result</th></tr>%s</table></body></html>`,
		color, status, passed, total, rows.String())
	os.WriteFile("report/index.html", []byte(page), 0o644)

	fmt.Printf("go-test: %d/%d passed, %d failed\n", passed, total, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
