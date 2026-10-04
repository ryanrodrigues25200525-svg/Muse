# Muse: TypeScript + OpenTUI Port — Design

**Date:** 2026-10-04
**Status:** Draft for review
**Supersedes:** `plan.md` (original Go build plan, phases 1–8, all shipped)

## 1. Context

Muse is a macOS terminal controller for Apple Music: a Cobra CLI (`mu`), a Bubble Tea
dashboard, and an MCP stdio server, all talking to Music.app through AppleScript.
The Go implementation is ~7,900 lines across `cmd/` and `internal/`, builds clean,
passes `go vet`, and has passing tests for `cmd`, `art`, `lyrics`, `mcp`, `music`, `tui`.

Two problems prompted this work:

1. **The TUI layer is due for a port.** The dashboard is built on Bubble Tea v1,
   which is feature-complete, while OpenTUI (the Zig core + TypeScript bindings
   that OpenCode ships in production) is where active TUI work for this ecosystem
   is happening. It also unlocks native album-art rendering and mouse support,
   which the Bubble Tea version could not do.
2. **The commands are not trustworthy.** Auditing the surface on a live machine
   found real defects (Section 6). The CLI also leaks raw AppleScript error text
   to users, and the binary installed at `/opt/homebrew/bin/mu` is a stale build
   missing `config` and the `queue` subcommands — so commands users expect simply
   do not exist.

The port is therefore **both** a rewrite and a repair. The repair is not incidental:
each command must be verified against a live Music.app, and that verification
becomes the behavioral contract the TypeScript implementation must match.

## 2. Goals

- Port all 24 existing commands, plus `mu lyrics` and `mu art` (Section 6, item 7),
  the mini display, the MCP stdio server, and the dashboard to TypeScript on OpenTUI,
  preserving observable behavior.
- Fix every defect found during the command audit, in the TypeScript implementation.
- Ship as one self-contained `mu` executable per macOS architecture via
  `bun build --compile`, keeping the Homebrew install path working.
- Leave a permanent regression net so AppleScript breakage is caught by tests, not
  by users.

## 3. Non-goals

- No new Apple Music features. Port and repair only. (Two dead-code commands,
  `lyrics` and `art`, become reachable — see Section 6, item 7.)
- No change to the `mu` command surface, flag names, or JSON output shapes beyond
  what bug fixes require.
- No cross-platform support. macOS only, as today.
- No MCP tool additions beyond parity.

## 4. Key decisions

### 4.1 Runtime and distribution: Bun, compiled to a single binary

`bun build --compile` embeds OpenTUI's native library, parser worker, and Tree-sitter
WASM directly into the executable, with no asset extraction and no `OTUI_ASSET_ROOT`.
This is documented Bun behavior, so the single-binary plan survives the port.

- Bun 1.3.0+ required (local toolchain: 1.4.2). Node is not used; Node support
  requires 26.4+ with `--experimental-ffi` and a manual asset-extraction prelude.
- **Per-architecture binaries, not universal.** `bun build --compile --target=bun-darwin-arm64`
  and `--target=bun-darwin-x64` each embed their own Bun runtime; merging them with
  `lipo` is untested for Bun-compiled executables. Release publishes both archives
  and the Homebrew formula selects by `Hardware::Cpu.arm64`.
- Version injection switches from Go's `-ldflags -X` to `bun build --define`.

### 4.2 UI layer: `@opentui/core` directly, not the React bindings

The Go dashboard is a `Model`/`Update`/`View` loop, which maps directly onto
OpenTUI Core's imperative renderable tree. Porting through a React reconciler would
add a translation layer for no benefit and enlarge the binary. `@opentui/keymap`
handles key bindings and command routing, replacing hand-rolled key switches.

This is the cheapest path to behavioral parity, which is the goal. If the dashboard
is later redesigned rather than ported, revisit this decision.

### 4.3 Structure: layered rewrite in place, Go retained until parity

The Go code stays in the tree, untouched, until the TypeScript version reaches
verified parity. Its value during the project is as an **oracle**: a second
implementation to diff live behavior against while auditing. It is removed in one
final commit.

### 4.4 CLI framework: Commander

Commander covers nested subcommands (`playlist play|export|import`, `queue add|clear|move`,
`config get|set|list|path`), boolean flags, help output, and exit codes. Shell
completions are generated from our own command registry (Section 5.5) rather than
borrowed from the framework, because the Go version's completions are part of its
documented interface.

## 5. Architecture

### 5.1 Module layout

```
src/
  main.ts                  entry: parse argv, dispatch (default -> dashboard)
  version.ts               MUSE_VERSION, injected at build time
  applescript/
    runner.ts              ScriptRunner interface + osascript implementation
    escape.ts              AppleScript string/number escaping
    errors.ts              AppleScriptError + message -> friendly-hint mapping
  music/
    types.ts               TrackInfo, TrackStats, LibraryStats, PlaylistInfo, DJEntry
    playback.ts            play/pause/toggle/next/prev/stop/seek/volume/shuffle/repeat/love
    now-playing.ts         NowPlaying, GetTrackStats
    library.ts             search, searchFiltered, recent, top, library stats, artwork
    playlists.ts           list, categorize, tracks, create, add, remove, play, export, import
    queue.ts               Music player-queue reads + persisted command queue
  cli/
    registry.ts            command tree definition (single source of truth)
    root.ts                Commander wiring, root command -> dashboard
    commands/              one module per command group, mirroring cmd/commands.go
    output.ts              human-readable vs --json rendering
    completion.ts          bash/zsh/fish script generation from registry.ts
  config/config.ts         ~/.config/muse/config.json
  lyrics/lyrics.ts         lrclib lookup + timed-line parsing
  art/art.ts               artwork path resolution, sizing, temp-file lifecycle
  tui/
    app.ts                 createCliRenderer, root tree, lifecycle/cleanup
    model.ts               state + update(), ported from dashboard.go
    views/                 nowPlaying, playlists, search, lyrics, queue, help
    keymap.ts              key bindings via @opentui/keymap
  mcp/server.ts            MCP stdio server, tool registry
```

Dependencies: `@opentui/core`, `@opentui/keymap`, `commander`,
`@modelcontextprotocol/sdk`, `zod`. Tests run on `bun test`. No other runtime deps.

### 5.2 The AppleScript transport is an injectable interface

This is the most important structural decision in the port:

```ts
export interface ScriptRunner {
  run(script: string): Promise<string>;
}

export const osascriptRunner: ScriptRunner = {
  run: (script) => spawn("osascript", ["-e", script]) /* trimmed stdout */
};
```

Every `music/*` module takes a `ScriptRunner` (defaulting to `osascriptRunner`).
Two consequences:

- **Tests assert the exact AppleScript emitted** without touching Music.app. This
  is how the audit's findings become a permanent regression net — a future macOS
  or Music.app change that breaks a script fails a unit test, not a user's session.
- **Live audit and test share one code path.** A defect found against a real
  Music.app is fixed once and covered by a test.

### 5.3 Typed errors replace leaked AppleScript text

Go returns osascript's combined output verbatim, so a user sees:

```
Error: applescript error: exit status 1 (output: execution error: Music got an error:
Can't get container of current track. (-1728))
```

`applescript/errors.ts` parses the osascript number and message into a typed
`AppleScriptError`, then maps known conditions to actionable messages:

| Condition | User-facing message |
| --- | --- |
| `-1728` while reading current track, player stopped | `Nothing is playing. Start playback with: mu play` |
| Not running / launch failure | `Music.app isn't running and couldn't be launched` |
| Automation permission denied (`-1743`) | Point at System Settings → Privacy & Security → Automation |
| Playlist not found | `No playlist named "X" found` |

Unrecognized AppleScript errors keep their number and message but lose the
`applescript error: exit status 1 (output: …)` wrapper noise. Every CLI command
handles this type; no command prints raw osascript output again.

### 5.4 Data flow

```
argv ──> cli/registry ──> music/* ──> ScriptRunner ──> osascript ──> Music.app
                            │                              │
                            └── parses delimited output ◀──┘
                                     │
                              cli/output ──> stdout (human | --json)
                                     │
                              tui/model ──> OpenTUI renderables
                              mcp/server ──> JSON-RPC over stdio
```

Music.app output is parsed with `\x1f` (unit separator) delimiters rather than
human-readable text, matching the existing `recordSep` constant, so track titles
containing newlines or commas cannot corrupt a record.

### 5.5 Completions

`cli/registry.ts` is the single source of truth for the command tree. Completion
scripts for bash, zsh, and fish are generated by walking that registry, so a new
command automatically appears in completions.

## 6. Audit findings — the repair scope

Found by reading the Go source and probing a live Music.app. Each is a
**hypothesis to confirm** during the audit phase; the fix lands in TypeScript.

1. **`queue add`/`clear`/`move` cannot work across processes.** The DJ queue is a
   package-level Go slice (`internal/music/queue.go:16`). `mu queue add "x"` reports
   "Queued… (1 in queue)" and the entry vanishes at exit. `mu queue` meanwhile reads
   Music's *player* queue — a different list. Fix: persist the command queue to
   disk (alongside `config.json`) and make the meanings distinct in help text.

2. **`repeat` cycles the wrong property.** `CycleRepeat` reads `song repeat as
   string`, which yields `"true"`/`"false"`, then compares against
   `"off"`/`"all"`/`"one"` — the values of `repeat mode`. The `off` branch is
   unreachable and `set song repeat to all` assigns a non-boolean. Confirmed by
   `mu now --json` returning `"repeat": ""`. Fix: use `repeat mode`, and surface
   the current mode in `now`.

3. **Raw AppleScript errors reach users.** Section 5.3.

4. **The installed binary is stale.** `/opt/homebrew/bin/mu` predates `config` and
   the `queue` subcommands. Users see "unknown command" for commands the README
   documents. Fix: reinstall; `doctor` additionally reports the version of the
   binary found on `PATH` so drift is visible.

5. **README and CLI disagree on `playlist`.** README documents `mu playlist "Gym"`;
   the command requires `mu playlist play "Gym"`. Decide one and align docs and CLI.

6. **`sleep` units are undocumented.** `mu sleep 30` means 30 minutes; neither
   command help nor README says so.

7. **`lyrics` and `art` exist but are unreachable from the CLI.** `internal/lyrics`
   and `internal/art` are reachable only from MCP and the TUI. Expose `mu lyrics`
   and `mu art` so the ported code is not shipped as dead code.

8. **`commands.go` import order is not gofmt-clean** (`syscall` before `runtime`).
   Cosmetic; moot once Go is deleted.

Each confirmed finding gets: a failing test at the `ScriptRunner` boundary, the fix,
and a live re-verification.

## 7. Dashboard port

The Bubble Tea model becomes an explicit state object plus a `update(event)` function,
mirroring the Go `Model`/`Update` structure so behavior is auditable line-by-line
against `internal/tui/dashboard.go`.

- `createCliRenderer()` owns the terminal; `tui/app.ts` builds the root renderable
  tree and guarantees cleanup on every exit path (including errors), matching
  Bubble Tea's teardown.
- Views: now-playing with progress, playlists browser, search, lyrics, queue, help
  footer. Ported one view at a time against the Go view functions.
- Keybindings preserved exactly: `Space` toggle, `n`/`p` next/prev, `s` shuffle,
  `r` repeat, `l` love, `[`/`]` seek, `+`/`-` volume, `/` search, `q` quit.
- **New capability, enabled by the port:** album artwork rendered via OpenTUI's
  `Image` renderable in the now-playing view. Additive only; the text layout is
  unchanged when artwork is unavailable, so no existing behavior regresses.

## 8. MCP server

The Go server is a hand-rolled JSON-RPC implementation over stdio. The port uses the
official `@modelcontextprotocol/sdk`, which handles protocol framing and validation.
Tool names, inputs, and JSON text return shapes are preserved exactly — agents
depending on them must not break. Tool count and behavior parity are verified by a
test that asserts the advertised tool list.

## 9. Build, release, distribution

- `Makefile`: `build` (bun run + compile), `test` (`bun test`), `lint`, `typecheck`,
  `install` (to `~/.local/bin`), `compile:arm64` / `compile:x64`.
- **CI** (`.github/workflows/ci.yml`): `bun install`, `typecheck`, `lint`, `bun test`
  on macOS arm64 and x64. AppleScript-touching tests are skipped in CI (no Music.app
  on runners); they run locally via `bun run test:live`.
- **Release workflow** replaces GoReleaser: build both architectures, checksum,
  upload archives, publish a draft release. Homebrew formula in the existing tap
  selects the archive by architecture.

## 10. Testing strategy

| Layer | Approach |
| --- | --- |
| AppleScript builders | Unit tests asserting emitted script text, via a fake `ScriptRunner` |
| Output parsers | Fixture tests over delimited records, including titles with newlines/commas/quotes |
| Commands | One test per command asserting exit code, stdout, and stderr text |
| Dashboard | OpenTUI's in-memory renderer for view snapshots and key handling |
| MCP | Assert the advertised tool list and one call/response per tool family |
| Live | `bun run test:live` drives real Music.app; run manually during the audit |

TDD throughout: a failing test first, then the fix.

## 11. Migration and cutover

Phased, with a working artifact at the end of each phase:

1. **Foundation** — Bun project, tsconfig, lint, test harness, `applescript/` layer
   with fake runner, install the OpenTUI agent skill (`npx skills add anomalyco/opentui
   --skill opentui`) so the port uses documented APIs.
2. **Audit** — confirm each Section 6 hypothesis against live Music.app; record
   actual vs. expected behavior per command. This is the repair backlog.
3. **Transport + config + lyrics + art** — the foundation every command needs.
4. **Commands, in dependency order** — playback → now/stats → search → playlists →
   queue → the rest. Each command: failing test, implementation, live verify.
5. **Dashboard** — port views, then keymap, then artwork.
6. **MCP server** — tool-by-tool parity against the Go server's advertised list.
7. **Distribution** — compile both architectures, rewrite the Homebrew formula and
   release workflow, reinstall locally over `/opt/homebrew/bin/mu`.
8. **Cutover** — delete Go, `go.mod`, `.goreleaser.yaml`; update README; tag a release.

## 12. Risks

| Risk | Mitigation |
| --- | --- |
| OpenTUI API differs from assumed shape | Phase 1 installs the official OpenTUI skill; docs are authoritative |
| Per-arch binaries disappoint users wanting one universal download | Homebrew hides it; release notes explain. A `lipo` universal build is attempted only if per-arch lands cleanly |
| Live Music.app behavior varies with library contents | Audit uses real data; `doctor` gains checks for the properties that proved fragile (e.g. `repeat mode` readability) |
| Audit balloons beyond the port's budget | Findings are fixed in TypeScript as commands are ported, never as a separate pre-pass; scope is capped at the 24 commands |
| Bun-compiled binary is unsigned and macOS quarantines it | Ad-hoc sign in the release workflow; document `xattr -d com.apple.quarantine` |

## 13. Success criteria

- All 24 ported commands plus `mu lyrics` and `mu art`, the mini display, the
  dashboard, and the MCP server present and behaving as documented.
- Every confirmed Section 6 finding fixed, each with a regression test.
- No command prints raw AppleScript error text.
- `make build` produces a working `mu` for the host architecture; both release
  artifacts verified on their target architecture.
- `brew install muse` and `mu mcp` work from the compiled binary.
- Go source removed; `README.md` reflects the TypeScript implementation.