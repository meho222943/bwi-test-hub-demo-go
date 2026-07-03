// reportgen reads `go test -json` output on stdin and writes
// report/junit.xml + a polished, self-contained report/index.html.
// All-Go, standard library only, so the whole project runs with just the
// Go toolchain.
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
	Action  string  `json:"Action"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

func main() {
	// per test: result action + elapsed seconds + accumulated output lines
	results := map[string]string{}
	elapsed := map[string]float64{}
	outputs := map[string][]string{}

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
			elapsed[e.Test] = e.Elapsed
		case "output":
			line := strings.TrimSpace(e.Output)
			if line != "" {
				outputs[e.Test] = append(outputs[e.Test], line)
			}
		}
	}

	names := make([]string, 0, len(results))
	for n := range results {
		names = append(names, n)
	}
	sort.Strings(names)

	total, failed := len(names), 0
	var totalElapsed float64
	for _, n := range names {
		totalElapsed += elapsed[n]
		if results[n] == "fail" {
			failed++
		}
	}
	passed := total - failed

	if err := os.MkdirAll("report", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// failure message: the most informative output line for a failed test
	// (skip go's "--- FAIL" / "=== RUN" framing; fall back to a default).
	failMessage := func(name string) string {
		for i := len(outputs[name]) - 1; i >= 0; i-- {
			l := outputs[name][i]
			if strings.HasPrefix(l, "--- FAIL") || strings.HasPrefix(l, "=== ") ||
				strings.HasPrefix(l, "FAIL") || strings.HasPrefix(l, "PASS") {
				continue
			}
			return l
		}
		return "test failed"
	}

	// JUnit XML
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<testsuite name="go-test" tests="%d" failures="%d" time="%.3f">`+"\n",
		total, failed, totalElapsed)
	for _, n := range names {
		if results[n] == "fail" {
			fmt.Fprintf(&b, "  <testcase classname=\"go-test\" name=\"%s\" time=\"%.3f\">\n",
				html.EscapeString(n), elapsed[n])
			fmt.Fprintf(&b, "    <failure message=\"%s\"/>\n", html.EscapeString(failMessage(n)))
			b.WriteString("  </testcase>\n")
		} else {
			fmt.Fprintf(&b, "  <testcase classname=\"go-test\" name=\"%s\" time=\"%.3f\"/>\n",
				html.EscapeString(n), elapsed[n])
		}
	}
	b.WriteString("</testsuite>\n")
	os.WriteFile("report/junit.xml", []byte(b.String()), 0o644)

	// HTML
	status, scls := "PASSED", "ok"
	if failed > 0 {
		status, scls = "FAILED", "no"
	}
	pct := 0
	if total > 0 {
		pct = passed * 100 / total
	}
	var rows strings.Builder
	for _, n := range names {
		if results[n] == "fail" {
			fmt.Fprintf(&rows,
				"<tr><td class=\"name\">%s</td><td><span class=\"chip no\">&#10007; failed</span><div class=\"msg\">%s</div></td></tr>",
				html.EscapeString(n), html.EscapeString(failMessage(n)))
		} else {
			fmt.Fprintf(&rows,
				"<tr><td class=\"name\">%s</td><td><span class=\"chip ok\">&#10003; passed</span></td></tr>",
				html.EscapeString(n))
		}
	}
	page := fmt.Sprintf(reportTemplate,
		scls, status, passed, total, failed, totalElapsed, pct, rows.String())
	os.WriteFile("report/index.html", []byte(page), 0o644)

	fmt.Printf("go-test: %d/%d passed, %d failed\n", passed, total, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

// reportTemplate order: title-status, pill-class, pill-status, passed,
// total, failed, duration, pct, rows.
const reportTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>go test &mdash; Test Report</title>
<style>
:root{--bg:#0b1120;--panel:#111a2e;--panel2:#0f1728;--border:#1e2a44;--text:#e6ecf7;--muted:#8ea0c0;--ok:#22c55e;--no:#ef4444}
*{box-sizing:border-box}
body{margin:0;font-family:ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;background:radial-gradient(1200px 600px at 20%% -10%%,#16223f 0%%,var(--bg) 55%%);color:var(--text);padding:2.5rem 1.25rem;min-height:100vh}
.wrap{max-width:920px;margin:0 auto}
.head{display:flex;align-items:center;gap:1rem;flex-wrap:wrap;justify-content:space-between;margin-bottom:1.75rem}
.title h1{margin:0;font-size:1.6rem;letter-spacing:-.02em}
.title p{margin:.35rem 0 0;color:var(--muted);font-size:.95rem}
.pill{display:inline-flex;align-items:center;gap:.5rem;padding:.5rem 1rem;border-radius:999px;font-weight:700;font-size:.95rem;color:#fff}
.pill.ok{background:linear-gradient(135deg,#16a34a,#22c55e);box-shadow:0 6px 20px -6px rgba(34,197,94,.55)}
.pill.no{background:linear-gradient(135deg,#dc2626,#ef4444);box-shadow:0 6px 20px -6px rgba(239,68,68,.55)}
.dot{width:.6rem;height:.6rem;border-radius:50%%;background:#fff}
.cards{display:grid;grid-template-columns:repeat(4,1fr);gap:.9rem;margin-bottom:1.5rem}
.card{background:linear-gradient(180deg,var(--panel),var(--panel2));border:1px solid var(--border);border-radius:14px;padding:1rem 1.1rem}
.card .k{color:var(--muted);font-size:.72rem;text-transform:uppercase;letter-spacing:.08em}
.card .v{font-size:1.7rem;font-weight:700;margin-top:.25rem}
.card.ok .v{color:var(--ok)}.card.no .v{color:var(--no)}
.bar{height:.5rem;border-radius:999px;background:#1e2a44;overflow:hidden;margin-bottom:1.75rem}
.bar>i{display:block;height:100%%;background:linear-gradient(90deg,#22c55e,#4ade80)}
table{width:100%%;border-collapse:collapse;background:var(--panel);border:1px solid var(--border);border-radius:14px;overflow:hidden}
th,td{padding:.8rem 1rem;text-align:left;font-size:.92rem;vertical-align:top}
thead th{background:#0e1626;color:var(--muted);font-weight:600;text-transform:uppercase;letter-spacing:.06em;font-size:.72rem}
tbody tr{border-top:1px solid var(--border)}
tbody tr:hover{background:#0e1830}
td.name{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
.chip{display:inline-flex;align-items:center;gap:.4rem;padding:.2rem .6rem;border-radius:999px;font-size:.78rem;font-weight:600}
.chip.ok{color:#bbf7d0;background:#14351f;border:1px solid #1f5133}
.chip.no{color:#fecaca;background:#3a1618;border:1px solid #5b2327}
.msg{color:#fca5a5;font-family:ui-monospace,monospace;font-size:.82rem;margin-top:.35rem}
.foot{color:var(--muted);font-size:.82rem;margin-top:1.25rem;display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem}
</style>
</head>
<body>
<div class="wrap">
  <div class="head">
    <div class="title"><h1>go test</h1><p>Go 1.23 &middot; testing &middot; BWI Test Hub demo</p></div>
    <span class="pill %s"><span class="dot"></span>%s</span>
  </div>
  <div class="cards">
    <div class="card"><div class="k">Passed</div><div class="v" style="color:var(--ok)">%d</div></div>
    <div class="card"><div class="k">Total</div><div class="v">%d</div></div>
    <div class="card no"><div class="k">Failed</div><div class="v">%d</div></div>
    <div class="card"><div class="k">Duration</div><div class="v">%.2fs</div></div>
  </div>
  <div class="bar"><i style="width:%d%%"></i></div>
  <table>
    <thead><tr><th>Test</th><th>Result</th></tr></thead>
    <tbody>%s</tbody>
  </table>
  <div class="foot"><span>Runtime: go-1 (container)</span><span>go test -json</span></div>
</div>
</body>
</html>`
