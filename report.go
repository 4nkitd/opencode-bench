package main

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

type modelStat struct {
	Model     string
	Passed    int
	Total     int
	CostUSD   float64
	TokensIn  int64
	TokensOut int64
	TimeS     float64
}

func buildHTMLReport(results []Result, stamp string) string {
	scenarioSet := map[string]bool{}
	statsByModel := map[string]*modelStat{}
	cell := map[string]Result{}
	for _, r := range results {
		scenarioSet[r.Scenario] = true
		s, ok := statsByModel[r.Model]
		if !ok {
			s = &modelStat{Model: r.Model}
			statsByModel[r.Model] = s
		}
		s.Total++
		if r.Passed {
			s.Passed++
		}
		s.CostUSD += r.CostUSD
		s.TokensIn += r.TokensIn
		s.TokensOut += r.TokensOut
		s.TimeS += r.DurationS
		cell[r.Model+"|"+r.Scenario] = r
	}

	var scenarios []string
	for s := range scenarioSet {
		scenarios = append(scenarios, s)
	}
	sort.Strings(scenarios)

	var stats []*modelStat
	for _, s := range statsByModel {
		stats = append(stats, s)
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Passed != stats[j].Passed {
			return stats[i].Passed > stats[j].Passed
		}
		if stats[i].CostUSD != stats[j].CostUSD {
			return stats[i].CostUSD < stats[j].CostUSD
		}
		return stats[i].TimeS < stats[j].TimeS
	})

	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8">
<title>opencode-bench report ` + stamp + `</title>
<style>
body{font-family:-apple-system,Segoe UI,sans-serif;margin:2rem auto;max-width:1100px;padding:0 1rem;background:#0f1115;color:#e6e6e6}
h1{font-size:1.4rem}h2{font-size:1.1rem;margin-top:2.2rem}
table{border-collapse:collapse;width:100%;margin:.8rem 0;font-size:.88rem}
th,td{border:1px solid #2a2e3a;padding:.45rem .6rem;text-align:left}
th{background:#1a1e28}
tr:nth-child(even){background:#151924}
.pass{color:#4ade80;font-weight:600}.fail{color:#f87171;font-weight:600}
.rank1{background:#14321f}
.muted{color:#8b93a7;font-size:.8rem}
details{margin:.5rem 0}summary{cursor:pointer;color:#8b93a7}
pre{background:#161a24;border:1px solid #2a2e3a;padding:.6rem;overflow-x:auto;font-size:.75rem;white-space:pre-wrap}
.bar{background:#2a2e3a;border-radius:4px;height:10px;width:120px;display:inline-block;vertical-align:middle;margin-right:.5rem}
.bar>i{display:block;height:100%;border-radius:4px;background:#4ade80}
</style></head><body>
<h1>opencode-bench report</h1>
<p class="muted">run ` + stamp + ` &middot; each cell = fresh git repo, model driven via <code>opencode run</code>, verified by hidden tests</p>
<h2>Leaderboard</h2>
<table><tr><th>#</th><th>Model</th><th>Pass rate</th><th>Total cost</th><th>Tokens in/out</th><th>Total agent time</th></tr>
`)
	for i, s := range stats {
		pct := 0
		if s.Total > 0 {
			pct = s.Passed * 100 / s.Total
		}
		cls := ""
		if i == 0 {
			cls = ` class="rank1"`
		}
		fmt.Fprintf(&b, `<tr%s><td>%d</td><td>%s</td><td><span class="bar"><i style="width:%d%%"></i></span>%d/%d (%d%%)</td><td>$%.4f</td><td>%s / %s</td><td>%.0fs</td></tr>
`,
			cls, i+1, html.EscapeString(s.Model), pct, s.Passed, s.Total, pct, s.CostUSD, fmtTok(s.TokensIn), fmtTok(s.TokensOut), s.TimeS)
	}
	b.WriteString("</table>\n<h2>Matrix</h2>\n<table><tr><th>Model</th>")
	for _, sc := range scenarios {
		fmt.Fprintf(&b, "<th>%s</th>", html.EscapeString(sc))
	}
	b.WriteString("</tr>\n")
	for _, s := range stats {
		fmt.Fprintf(&b, "<tr><td>%s</td>", html.EscapeString(s.Model))
		for _, sc := range scenarios {
			r, ok := cell[s.Model+"|"+sc]
			if !ok {
				b.WriteString("<td>-</td>")
				continue
			}
			mark, cls := "FAIL", "fail"
			if r.Passed {
				mark, cls = "PASS", "pass"
			}
			fmt.Fprintf(&b, `<td><span class="%s">%s</span><br><span class="muted">%.0fs &middot; $%.4f</span></td>`, cls, mark, r.DurationS, r.CostUSD)
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n<h2>Failure details</h2>\n")
	nFail := 0
	for _, r := range results {
		if r.Passed {
			continue
		}
		nFail++
		fmt.Fprintf(&b, "<details><summary>%s &middot; %s</summary>",
			html.EscapeString(r.Model), html.EscapeString(r.Scenario))
		if r.AgentError != "" {
			fmt.Fprintf(&b, "<p class=\"muted\">agent error</p><pre>%s</pre>", html.EscapeString(r.AgentError))
		}
		if r.VerifyLog != "" {
			fmt.Fprintf(&b, "<p class=\"muted\">verify output</p><pre>%s</pre>", html.EscapeString(r.VerifyLog))
		}
		b.WriteString("</details>\n")
	}
	if nFail == 0 {
		b.WriteString("<p class=\"muted\">no failures</p>\n")
	}
	b.WriteString("</body></html>\n")
	return b.String()
}

func fmtTok(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprintf("%d", n)
}
