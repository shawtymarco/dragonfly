# Dragonfly (shawtymarco fork)

Fork of `df-mc/dragonfly`. Keep the Go module path `github.com/df-mc/dragonfly`.
CastleOnline uses this via `replace`. Sync from `df-mc/dragonfly` yourself. Do not add a gophertunnel replace.

## Commits

Match upstream df-mc commit messages. When committing or proposing a commit, always state the **scope** and the full **message**.

Format: `scope: description`

- **scope**: Go package or file the change lives in, relative to the repo root. Prefer the package (`server/session`, `server/world`, `server`, `server/block`). Use a file (`block/composter.go`, `world/weather.go`) when the change is confined to one file.
- **description**: concise, imperative. Usually starts with a verb (`fix`, `add`, `allow`, `map`). No trailing period.
- **issue**: append `(df-mc#N)` or `(#N)` only when there is a matching upstream issue or PR. Do not invent numbers.

Examples:

- `server/session: map adventure and spectator to Bedrock game types`
- `server: add stdin console commands and per-world TPS stats`
- `server/world: snapshot chunk and entity counts atomically to avoid status deadlock`
- `server: allow disabling the nether and end dimensions`
- `server/block: fix dropping the single variant of blocks that can stack (df-mc#1423)`
- `item/recipe: Fix recipe book visibility, add shulker box and multi recipes (#1386)`
