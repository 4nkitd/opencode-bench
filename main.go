package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Scenario struct {
	Name       string `json:"name"`
	Prompt     string `json:"prompt"`
	VerifyCmd  string `json:"verify_cmd"`
	TimeoutMin int    `json:"timeout_min"`
}

type Result struct {
	Scenario   string  `json:"scenario"`
	Model      string  `json:"model"`
	Passed     bool    `json:"passed"`
	DurationS  float64 `json:"duration_s"`
	CostUSD    float64 `json:"cost_usd"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	VerifyLog  string  `json:"verify_log,omitempty"`
	AgentError string  `json:"agent_error,omitempty"`
	WorkDir    string  `json:"work_dir"`
}

func main() {
	models := flag.String("models", "", "comma-separated provider/model list (required)")
	only := flag.String("only", "", "run only these scenarios (comma-separated)")
	keep := flag.Bool("keep", false, "keep work dirs after run")
	out := flag.String("out", "results", "results output dir")
	flag.Parse()

	if *models == "" {
		fmt.Fprintln(os.Stderr, "usage: opencode-bench -models provider/model[,provider/model...]")
		os.Exit(1)
	}

	root, err := os.Getwd()
	must(err)
	scenarios, err := loadScenarios(filepath.Join(root, "scenarios"), *only)
	must(err)
	if len(scenarios) == 0 {
		fmt.Fprintln(os.Stderr, "no scenarios found")
		os.Exit(1)
	}

	must(os.MkdirAll(*out, 0o755))
	var results []Result
	for _, model := range strings.Split(*models, ",") {
		model = strings.TrimSpace(model)
		for _, sc := range scenarios {
			fmt.Printf("== %s | %s\n", model, sc.Name)
			r := runOne(root, sc, model, *keep)
			results = append(results, r)
			status := "FAIL"
			if r.Passed {
				status = "PASS"
			}
			fmt.Printf("   %s  %.0fs  $%.4f\n", status, r.DurationS, r.CostUSD)
		}
	}

	stamp := time.Now().Format("20060102-150405")
	jsonPath := filepath.Join(*out, "run-"+stamp+".json")
	b, _ := json.MarshalIndent(results, "", "  ")
	must(os.WriteFile(jsonPath, b, 0o644))
	fmt.Println()
	printTable(results)
	fmt.Println("\nresults:", jsonPath)
}

func loadScenarios(dir, only string) ([]Scenario, error) {
	filter := map[string]bool{}
	for _, n := range strings.Split(only, ",") {
		if n = strings.TrimSpace(n); n != "" {
			filter[n] = true
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Scenario
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if len(filter) > 0 && !filter[e.Name()] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name(), "scenario.json"))
		if err != nil {
			return nil, err
		}
		var sc Scenario
		if err := json.Unmarshal(b, &sc); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		sc.Name = e.Name()
		if sc.TimeoutMin == 0 {
			sc.TimeoutMin = 10
		}
		out = append(out, sc)
	}
	return out, nil
}

func runOne(root string, sc Scenario, model string, keep bool) Result {
	r := Result{Scenario: sc.Name, Model: model}
	scDir := filepath.Join(root, "scenarios", sc.Name)

	work, err := os.MkdirTemp("", "ocbench-"+sc.Name+"-")
	if err != nil {
		r.AgentError = err.Error()
		return r
	}
	r.WorkDir = work
	if !keep {
		defer os.RemoveAll(work)
	}

	if err := copyTree(filepath.Join(scDir, "seed"), work); err != nil {
		r.AgentError = "seed copy: " + err.Error()
		return r
	}
	gitRun(work, "init", "-q")
	gitRun(work, "add", "-A")
	gitRun(work, "commit", "-qm", "seed", "--no-gpg-sign")

	start := time.Now()
	cmd := exec.Command("opencode", "run", "-m", model, "--dir", work, "--auto", "--title", "bench:"+sc.Name, sc.Prompt)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "OPENCODE_DISABLE_LSP_DOWNLOAD=true")
	done := make(chan error, 1)
	var agentOut strings.Builder
	cmd.Stdout = &agentOut
	cmd.Stderr = &agentOut
	if err := cmd.Start(); err != nil {
		r.AgentError = err.Error()
		return r
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			r.AgentError = trim(agentOut.String(), 2000)
		}
	case <-time.After(time.Duration(sc.TimeoutMin) * time.Minute):
		_ = cmd.Process.Kill()
		r.AgentError = "timeout"
	}
	r.DurationS = time.Since(start).Seconds()

	if err := copyTree(filepath.Join(scDir, "hidden"), work); err != nil {
		r.AgentError = "hidden copy: " + err.Error()
		return r
	}

	vc := exec.Command("sh", "-c", sc.VerifyCmd)
	vc.Dir = work
	vout, verr := vc.CombinedOutput()
	r.Passed = verr == nil
	if !r.Passed {
		r.VerifyLog = trim(string(vout), 3000)
	}

	fillUsage(&r, work)
	return r
}

func fillUsage(r *Result, dir string) {
	home, _ := os.UserHomeDir()
	db, err := sql.Open("sqlite", filepath.Join(home, ".local/share/opencode/opencode.db"))
	if err != nil {
		return
	}
	defer db.Close()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		resolved = dir
	}
	row := db.QueryRow(`select cost, tokens_input+tokens_cache_read+tokens_cache_write, tokens_output+tokens_reasoning
		from session where directory in (?, ?) order by time_created desc limit 1`, dir, resolved)
	_ = row.Scan(&r.CostUSD, &r.TokensIn, &r.TokensOut)
}

func copyTree(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, strings.TrimSuffix(rel, ".hidden"))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func gitRun(dir string, args ...string) {
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=bench", "GIT_AUTHOR_EMAIL=bench@local",
		"GIT_COMMITTER_NAME=bench", "GIT_COMMITTER_EMAIL=bench@local")
	_ = c.Run()
}

func printTable(results []Result) {
	fmt.Printf("%-28s %-34s %-6s %8s %10s\n", "SCENARIO", "MODEL", "PASS", "TIME(s)", "COST($)")
	byModel := map[string][2]int{}
	for _, r := range results {
		p := "no"
		if r.Passed {
			p = "yes"
		}
		fmt.Printf("%-28s %-34s %-6s %8.0f %10.4f\n", r.Scenario, r.Model, p, r.DurationS, r.CostUSD)
		s := byModel[r.Model]
		s[1]++
		if r.Passed {
			s[0]++
		}
		byModel[r.Model] = s
	}
	fmt.Println()
	for m, s := range byModel {
		fmt.Printf("%s: %d/%d passed\n", m, s[0], s[1])
	}
}

func trim(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
