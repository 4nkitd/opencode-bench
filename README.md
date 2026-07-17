# opencode-bench

Personal AI-model benchmark built from my real OpenCode work (Paytring Go backend,
Shopify webhooks, cart-drawer JS, port-kill CLI, prod debugging). Every run drives
the model through OpenCode itself in a fresh git repo per scenario.

## Scenarios

| Scenario | Type | Based on |
|---|---|---|
| go-webhook-retry | bugfix | breeze.quick "200-on-error so Shopify never retries" incident |
| go-currency-market | implement | currency→country market mapping + minor-unit money formatting |
| js-cart-progress | implement | cart drawer free-shipping bar + formatMoney |
| go-portlist | implement | port-kill lsof -F pcn parser |
| go-coupon-sync-bug | debug | gift-sync stuck in-flight guard bug |

Each scenario dir: `scenario.json` (prompt + verify cmd), `seed/` (what the model
sees, git-initialized), `hidden/` (tests copied in AFTER the agent finishes — it
can't game them).

## Run

```sh
go build -o ocbench .
./ocbench -models anthropic/claude-fable-5,Paytring/DeepSeek-V4-Pro
./ocbench -models some/model -only go-portlist -keep   # single scenario, keep workdir
```

Per (model, scenario): temp dir → seed copied → `git init` + commit →
`opencode run -m <model> --dir <workdir> --auto "<prompt>"` → hidden tests copied
in → verify command → pass/fail, wall time, and cost/tokens pulled from
`~/.local/share/opencode/opencode.db` by workdir. Results printed as a table and
saved to `results/run-<ts>.json`.

Prompts are deliberately short and repos tiny to keep token burn low
(~one small task per scenario, 10 min timeout).
