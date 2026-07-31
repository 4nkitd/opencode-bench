package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ScenarioMeta struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	Lang       string `json:"lang"`
	Difficulty string `json:"difficulty"`
	Goal       string `json:"goal"`
	Signal     string `json:"signal"`
}

type Data struct {
	GeneratedAt string         `json:"generated_at"`
	Scenarios   []ScenarioMeta `json:"scenarios"`
	Runs        []Result       `json:"runs"`
}

// dead entries: local LoRA experiments and opencode-zen routes that never got a
// working tool-call loop. All scored 0% and just pad the board. Note these are
// exact IDs — the opencode-go/* equivalents are real results and stay.
var excludedModels = map[string]bool{
	"gemma4-lora/my-lora":     true,
	"emma4-lora/my-lora":      true,
	"opencode/kimi-k2.7-code": true,
	"opencode/minimax-m3":     true,
	"opencode/glm-5.2":        true,
}

func writeDataJSON(results []Result) string {
	kept := results[:0:0]
	for _, r := range results {
		if !excludedModels[r.Model] {
			kept = append(kept, r)
		}
	}
	results = kept

	var metas []ScenarioMeta
	entries, _ := os.ReadDir("scenarios")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join("scenarios", e.Name(), "scenario.json"))
		if err != nil {
			continue
		}
		var sc Scenario
		if json.Unmarshal(b, &sc) != nil {
			continue
		}
		metas = append(metas, ScenarioMeta{
			Name: e.Name(), Title: sc.Title, Lang: sc.Lang,
			Difficulty: sc.Difficulty, Goal: sc.Goal, Signal: sc.Signal,
		})
	}
	for i := range results {
		results[i].VerifyLog = trim(results[i].VerifyLog, 800)
		results[i].AgentError = trim(results[i].AgentError, 800)
	}
	d := Data{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Scenarios:   metas,
		Runs:        results,
	}
	must(os.MkdirAll("web", 0o755))
	b, _ := json.MarshalIndent(d, "", "  ")
	path := filepath.Join("web", "data.json")
	must(os.WriteFile(path, b, 0o644))
	return path
}
