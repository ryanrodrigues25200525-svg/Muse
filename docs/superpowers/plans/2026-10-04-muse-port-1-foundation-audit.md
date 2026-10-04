# Muse TypeScript Port — Plan 1: Foundation and Command Audit

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the Bun/TypeScript project with a tested AppleScript transport layer, then audit all 24 commands against a live Music.app to produce the repair backlog that Plans 2–4 execute against.

**Architecture:** Every AppleScript call in Muse goes through one injectable `ScriptRunner` interface. That seam lets tests assert the exact script text without touching Music.app, and turns every audit finding into a permanent regression test. This plan builds the seam and uses it to audit; it ports no commands yet.

**Tech Stack:** Bun 1.4.2, TypeScript (strict), `bun:test`, `osascript`. No runtime dependencies are added in this plan.

**Spec:** `docs/superpowers/specs/2026-10-04-muse-opentui-port-design.md`

**Downstream plans (not written yet, listed so interfaces stay stable):**
- Plan 2 — CLI parity: all 24 commands, mini display, completions, doctor.
- Plan 3 — Dashboard on `@opentui/core`, plus `mu` with no args.
- Plan 4 — MCP server, packaging/release, Go removal.

## Global Constraints

- Bun 1.3.0 or later. Local toolchain is 1.4.2. Node is not used.
- macOS only. Tests that shell out to `osascript` skip themselves on other platforms.
- TypeScript `strict: true`. No `any` in committed code.
- All AppleScript execution goes through `ScriptRunner`. No direct `Bun.spawn("osascript")` outside `src/applescript/runner.ts`.
- Records are delimited with `\x1f` (ASCII character 31). Never parse human-readable AppleScript prose.
- Escaping into AppleScript string literals replaces `\`, `"`, `\n`, and `\r`, matching Go's `escapeAS`.
- No user-facing command may print raw `osascript` output (spec §5.3).
- TDD: a failing test precedes every implementation. Commit after every task.

## Review Focus

Five conditions a person using Muse would reasonably expect to work, which the spec implies but no implementation task naturally tests. Each is pinned to a test in the task named below.

1. **Track and playlist names containing `|`, `"`, `\`, or newlines.** The live library contains names like `Sharma (ਸ਼ਰਮਾ) Music`, `💔😭`, and `🔥🔥❤️`. A naive delimiter split silently corrupts these into wrong titles. → Task 4
2. **Player stopped while reading the current track.** `mu queue` on a stopped player currently prints `Error: applescript: Music got an error: Can't get container of current track. (-1728)`. Expected: an actionable message telling the user to start playback. This is the exact live bug that motivated the audit. → Task 3
3. **Automation permission denied (`-1743`).** This machine has permission granted, so a live-only test can never catch it; it must be exercised through an injected error. Expected: a message pointing at System Settings → Privacy & Security → Automation. → Task 3
4. **Non-ASCII names survive parsing.** Go pads columns with `%-32s`, which misaligns any name containing wide or RTL characters, and the live library is full of them (`🇦🇪`, `العربية`, `Sharma (ਸ਼ਰਮਾ) Music`). Expected: the parser round-trips these names byte-identically (pinned here in Task 4); the column-padding half of this defect lands in Plan 2, which owns the formatters.
5. **Boundary values rejected, not clamped.** `mu volume 101`, `mu volume -1`, `mu volume abc`, and a 6-star rating must all be rejected with a clear message rather than silently clamped or wrapped. → Task 5 (the audit records what the Go CLI actually does; the validation itself is implemented in Plan 2)

## File Structure

| Path | Responsibility |
| --- | --- |
| `package.json` | Bun scripts: `test`, `test:live`, `typecheck`, `audit` |
| `tsconfig.json` | Strict TS config for Bun |
| `src/version.ts` | `MUSE_VERSION`, injected at build time |
| `src/applescript/runner.ts` | `ScriptRunner` interface and the only `osascript` spawn site |
| `src/applescript/errors.ts` | `AppleScriptError`, stderr parsing, user-facing message mapping |
| `src/applescript/escape.ts` | AppleScript string-literal escaping |
| `src/applescript/record.ts` | `\x1f` record and row splitting |
| `scripts/audit.ts` | Runs the Go binary as an oracle plus raw `osascript` probes |
| `docs/superpowers/audit/2026-10-04-command-audit.md` | The repair backlog |
| `test/applescript/*.test.ts` | Unit tests for the four `applescript/` modules |

---

### Task 1: Bun project skeleton and version module

**Files:**
- Create: `package.json`
- Create: `tsconfig.json`
- Create: `src/version.ts`
- Create: `test/version.test.ts`
- Modify: `.gitignore` (ignore `node_modules/`, `bun.lockb` is tracked; ignore `audit-out/`)

**Interfaces:**
- Produces: `MUSE_VERSION: string` from `src/version.ts`. Every later plan imports this; the release build injects a real value via `bun build --define`.

- [ ] **Step 1: Write the failing test**

`test/version.test.ts`:
```ts
import { expect, test } from "bun:test"
import { MUSE_VERSION } from "../src/version"

test("MUSE_VERSION is a non-empty string", () => {
  expect(typeof MUSE_VERSION).toBe("string")
  expect(MUSE_VERSION.length).toBeGreaterThan(0)
})

test("MUSE_VERSION defaults to dev when not injected at build time", () => {
  // Under bun test nothing defines MUSE_VERSION, so the default must survive.
  expect(MUSE_VERSION).toBe("dev")
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `bun test test/version.test.ts`
Expected: FAIL — `Cannot find module "../src/version"`.

- [ ] **Step 3: Implement `MUSE_VERSION` in `src/version.ts`**

One line: `export const MUSE_VERSION = "dev"` with a comment that release builds replace it via `bun build --define MUSE_VERSION='"x.y.z"'`. The literal must stay `"dev"` so Step 2's second assertion holds.

- [ ] **Step 4: Create `package.json`**

`name: "muse"`, `type: "module"`, `private: true`, `module`/`target` left to Bun. Scripts: `test` → `bun test`, `test:live` → `MUSE_LIVE=1 bun test`, `typecheck` → `tsc --noEmit`, `audit` → `bun run scripts/audit.ts`. No dependencies yet; Plan 2 adds `commander`, Plan 3 adds OpenTUI.

- [ ] **Step 5: Create `tsconfig.json`**

`strict: true`, `module: "ESNext"`, `moduleResolution: "bundler"`, `target: "ESNext"`, `types: ["bun-types"]`, `noEmit: true`, `skipLibCheck: true`. Add `node_modules/` and `audit-out/` to `.gitignore`.

- [ ] **Step 6: Run the test to verify it passes**

Run: `bun test test/version.test.ts && bun run typecheck`
Expected: 2 pass, 0 fail; typecheck exits 0.

- [ ] **Step 7: Install the Bun type definitions**

Run: `bun add -d @types/bun && bunx tsc --noEmit`
Expected: exits 0. If `@types/bun` conflicts with the `types` entry in Step 5, use `["bun"]` instead.

- [ ] **Step 8: Commit**

```bash
git add package.json tsconfig.json src/version.ts test/version.test.ts .gitignore
git commit -m "chore: scaffold Bun + TypeScript project with strict typing"
```

---

### Task 2: AppleScript transport

**Files:**
- Create: `src/applescript/runner.ts`
- Create: `test/applescript/runner.test.ts`

**Interfaces:**
- Produces:
  ```ts
  export interface ScriptRunner {
    run(script: string): Promise<string>
  }
  export function createOsascriptRunner(): ScriptRunner
  ```
  `run` resolves with trimmed stdout, and rejects with an `AppleScriptError` (Task 3 defines the class; declare the import then, or define the class in Task 3 and have this task throw a plain `Error` upgraded in Task 3). To keep this task self-contained and runnable, create `AppleScriptError` in Task 3 and in this task reject with an `Error` whose `message` is the trimmed stderr; Task 3's first test pins the upgrade.

  Decision, stated once so no implementer has to choose: **Task 2 defines `AppleScriptError` in `src/applescript/errors.ts` as a minimal class carrying `rawOutput`, and Task 3 extends it with parsing and message mapping.** Task 2 therefore creates `errors.ts` too.

- [ ] **Step 1: Write the failing tests**

`test/applescript/runner.test.ts` — three cases: a script that returns a string resolves with it trimmed; a script that raises an AppleScript error rejects with `AppleScriptError` whose `rawOutput` contains the osascript message; `createOsascriptRunner()` returns an object with a `run` function.

The first two shell out to real `osascript`, so guard the file with `const describeMac = process.platform === "darwin" ? describe : describe.skip`.

Assert against these exact scripts and expectations:
- `run('return "ok"')` resolves `"ok"`
- `run('return "  padded  "')` resolves `"padded"`
- `run('error "boom" number -1728')` rejects, and `err.rawOutput` includes `boom`

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bun test test/applescript/runner.test.ts`
Expected: FAIL — module `../src/applescript/runner` not found.

- [ ] **Step 3: Implement `createOsascriptRunner` in `src/applescript/runner.ts`**

```ts
export interface ScriptRunner {
  run(script: string): Promise<string>
}

export function createOsascriptRunner(): ScriptRunner
```

Use `Bun.spawn(["osascript", "-e", script], { stdout: "pipe", stderr: "pipe" })`, `await proc.exited`, then read both streams with `new Response(proc.stdout).text()`. On a nonzero exit, throw `new AppleScriptError(stderr.trim(), stderr.trim())`; otherwise resolve `stdout.trim()`. Create `src/applescript/errors.ts` in this task with the minimal shape Task 3 extends:

```ts
export class AppleScriptError extends Error {
  constructor(message: string, readonly rawOutput: string) {
    super(message)
    this.name = "AppleScriptError"
  }
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `bun test test/applescript/runner.test.ts`
Expected: 3 pass, 0 fail.

- [ ] **Step 5: Commit**

```bash
git add src/applescript/runner.ts src/applescript/errors.ts test/applescript/runner.test.ts
git commit -m "feat: injectable AppleScript ScriptRunner transport"
```

---

### Task 3: Typed errors and user-facing messages

**Files:**
- Modify: `src/applescript/errors.ts`
- Create: `test/applescript/errors.test.ts`

**Interfaces:**
- Consumes: `AppleScriptError` from Task 2.
- Produces:
  ```ts
  export function parseOsascriptError(stderr: string): AppleScriptError
  export function friendlyMessage(err: unknown): string
  ```
  `parseOsascriptError` returns an `AppleScriptError` carrying `osascriptNumber: number | null`, parsed from a trailing `(-?\d+)`; `friendlyMessage` accepts any thrown value and returns a message safe to print.

- [ ] **Step 1: Write the failing tests**

`test/applescript/errors.test.ts`, pinning the spec §5.3 table exactly:

| Input to `parseOsascriptError` | Expected `osascriptNumber` | Expected `friendlyMessage` |
| --- | --- | --- |
| `execution error: Music got an error: Can't get container of current track. (-1728)` | `1728` | `Nothing is playing. Start playback with: mu play` |
| `execution error: Not authorized to send Apple events to Music. (-1743)` | `1743` | `Not authorized to send Apple events to Music. Open System Settings → Privacy & Security → Automation and grant your terminal access to Music.app` |
| `execution error: whatever (-1234)` | `1234` | includes `whatever` and does not include `execution error:` |
| `command not found` | `null` | includes `command not found` |

Plus: `friendlyMessage(new Error("plain"))` returns `"plain"`, and `friendlyMessage` on an `AppleScriptError` never contains the substring `osascript error:`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bun test test/applescript/errors.test.ts`
Expected: FAIL — `parseOsascriptError` is not exported.

- [ ] **Step 3: Implement `parseOsascriptError` and `friendlyMessage`**

Extend `AppleScriptError` with `readonly osascriptNumber: number | null`, defaulting to `null` so Task 2's constructor call sites keep compiling.

Parse with a regex anchored to the final parenthesised number in the last non-empty line: `/\((-?\d+)\)\s*$/`. Strip a leading `execution error: ` / `script error: ` prefix from the message.

`friendlyMessage` dispatches on `err instanceof AppleScriptError`, returning the table row for `1728` (with no current track) and `1743`, and otherwise returning the message with the wrapper removed. For any non-`AppleScriptError` value, return `err instanceof Error ? err.message : String(err)`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `bun test test/applescript/errors.test.ts`
Expected: all pass.

- [ ] **Step 5: Run the whole suite and typecheck**

Run: `bun test && bun run typecheck`
Expected: all pass, typecheck exits 0.

- [ ] **Step 6: Commit**

```bash
git add src/applescript/errors.ts test/applescript/errors.test.ts
git commit -m "feat: typed AppleScript errors with user-facing messages"
```

---

### Task 4: Escaping and record parsing

**Files:**
- Create: `src/applescript/escape.ts`
- Create: `src/applescript/record.ts`
- Create: `test/applescript/escape.test.ts`
- Create: `test/applescript/record.test.ts`

**Interfaces:**
- Produces:
  ```ts
  // escape.ts
  export function escapeAppleScriptString(value: string): string
  // record.ts
  export const RECORD_SEP = "\x1f"
  export function parseRecord(output: string): string[]
  export function parseRows(output: string, sep?: string): string[][]
  ```

- [ ] **Step 1: Write the failing tests**

`test/applescript/escape.test.ts` — mirrors Go's `escapeAS` (`internal/music/controller.go:1968`):

| Input | Expected |
| --- | --- |
| `plain` | `plain` |
| `say "hi"` | `say \"hi\"` |
| `back\\slash` | `back\\\\slash` |
| `two\nlines` | `two lines` |
| `two\rlines` | `two lines` |

`test/applescript/record.test.ts`:

- `parseRecord("a\u001fb\u001fc")` → `["a", "b", "c"]`
- `parseRecord("")` → `[""]`
- `parseRecord("solo")` → `["solo"]`
- **Delimiter-in-content (Review Focus #1):** `parseRecord("Sharma (ਸ਼ਰਮਾ) Music\u001f🔥🔥❤️\u001fR&B")` → exactly three fields, with the emoji and Devanagari preserved intact
- **Newline safety:** `parseRecord("two\nlines\u001fb")` → `["two\nlines", "b"]`, proving newlines survive inside a field rather than being flattened
- `parseRows("a|b|c\nd|e|f", "|")` → `[["a","b","c"], ["d","e","f"]]`
- `parseRows("a|b|c\n\nd|e|f", "|")` → two rows, the blank line dropped
- **Non-ASCII integrity (Review Focus #4):** `parseRows("🇦🇪|العربية|/ghazal\n🇬🇧|UK|rap", "|")` → two rows of three fields each, byte-identical to input

- [ ] **Step 2: Run the tests to verify they fail**

Run: `bun test test/applescript/`
Expected: FAIL — modules not found.

- [ ] **Step 3: Implement `escapeAppleScriptString` in `src/applescript/escape.ts`**

Apply replacements in this exact order (backslash first, or the escapes re-escape each other): `\` → `\\`, `"` → `\"`, `\n` → space, `\r` → space.

- [ ] **Step 4: Implement `parseRecord` and `parseRows` in `src/applescript/record.ts`**

`parseRecord` splits on `RECORD_SEP`, which this module exports as `"\x1f"` to match Go's `recordSep` (`internal/music/controller.go:61`). `parseRows` splits output on `\n`, drops blank lines, and splits each remaining line on `sep ?? "|"` — the default matches Go's `parseTrackLines`. Neither function trims or normalizes field contents; leading and trailing spaces inside a title are meaningful.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `bun test test/applescript/`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add src/applescript/escape.ts src/applescript/record.ts test/applescript/escape.test.ts test/applescript/record.test.ts
git commit -m "feat: AppleScript escaping and record parsing"
```

---

### Task 5: Live command audit and repair backlog

**Files:**
- Create: `scripts/audit.ts`
- Create: `docs/superpowers/audit/2026-10-04-command-audit.md`

**Interfaces:**
- Consumes: `createOsascriptRunner` and `parseRecord` from Tasks 2–4; the Go oracle binary.
- Produces: `docs/superpowers/audit/2026-10-04-command-audit.md`, whose "Repair Backlog" table becomes the task list for Plan 2.

**This task intentionally mutates the user's Music.app.** They approved live playback driving. Start a track before probing, and leave playback paused rather than stopped when finished.

- [ ] **Step 1: Build the Go oracle binary**

Run: `go build -o audit-out/mu-go main.go`
Expected: exits 0. `audit-out/` is gitignored. This binary is the reference implementation the audit compares against.

- [ ] **Step 2: Write `scripts/audit.ts` with the command matrix**

The script must, for each entry in the matrix, run the oracle binary and capture stdout, stderr, and exit code into a JSON object, then print a summary table.

Command matrix — all 24 commands, each with the exact invocation the script uses:

| # | Command | Invocation | Notes |
| --- | --- | --- | --- |
| 1 | `play` | `play` | resumes playback |
| 2 | `pause` | `pause` | |
| 3 | `toggle` | `toggle` | |
| 4 | `next` | `next` | |
| 5 | `prev` | `prev` | |
| 6 | `stop` | `stop` | |
| 7 | `now` | `now`, `now --json` | |
| 8 | `volume` | `volume`, then `volume 0`, `volume 100`, `volume 101`, `volume -1`, `volume abc` | mutating; restore to the value read by the first call |
| 9 | `seek` | `seek +10`, `seek -5`, `seek 10` | third call tests the missing-sign path |
| 10 | `shuffle` | `shuffle toggle` | mutating; restore by toggling again if the audit enabled it |
| 11 | `repeat` | `repeat`, `repeat`, `repeat`, `repeat` | four calls walk the full cycle and expose a stuck value; restore to `off` |
| 12 | `playlist` | `playlist`, `playlist play "Drill"`, `playlist export "Drill" --format m3u`, `playlist export "Drill" --format json`, `playlist import "audit-tmp" /dev/null` | bare `playlist` prints help; `import` with `/dev/null` must fail cleanly without creating a playlist |
| 13 | `playlists` | `playlists`, `playlists --json` | |
| 14 | `search` | `search "a" --limit 3`, `search --artist "Drill" --limit 3`, `search` (no args) | third call tests the no-query error path |
| 15 | `love` | `love`, `love` | toggles twice to restore original loved state |
| 16 | `stats` | `stats`, `stats --json` | |
| 17 | `share` | `share` | overwrites the clipboard with the track name; note it in the report |
| 18 | `sleep` | `sleep 0`, `sleep -1`, `sleep abc` | **not** `sleep 1` — a positive value blocks for a full minute. The three invalid values return immediately and still prove validation |
| 19 | `queue` | `queue`, `queue --json`, `queue add "a"`, `queue clear`, `queue move 1 2` | run `add` and `queue` as separate processes to expose finding 1 |
| 20 | `mini` | `mini --help` | **not audited (blocking)** — `mini` loops until interrupted |
| 21 | `doctor` | `doctor` | |
| 22 | `mcp` | `mcp --help` | **not audited (blocking)** — serves stdio forever |
| 23 | `version` | `version` | |
| 24 | `config` | `config list`, `config path`, `config get art_size`, `config set art_size medium` | the `set` restores its own prior value |

Three commands get an explicit `not audited (blocking)` verdict rather than a guess: `mini` and `mcp` above, and `sleep` beyond validation. For each, record the entry's `--help` text as evidence the command is wired, mark the behavior `not audited (blocking)`, and note that Plan 2 audits it under a PTY. No other command may use that verdict without a stated reason.

Before the matrix, the script reads the current `volume`, `shuffle`, `repeat`, and `loved` state via the oracle's JSON output and restores each at the end, so the audit leaves the user's Music.app as it found it.

- [ ] **Step 3: Add the targeted hypothesis probes**

These answer the six spec §6 findings directly. Each probe runs through `createOsascriptRunner()`:

1. **Repeat mode (finding 2).** Read `repeat mode` and `song repeat` separately with `tell application "Music" to get {repeat mode, song repeat}`. Record both raw values. This settles whether `CycleRepeat`'s `song repeat as string` comparison can ever match `"off"`.
2. **Stopped-player error text (finding 3).** With playback stopped, read `current track` and capture the exact osascript error, confirming the `-1728` message the friendly-message table must match.
3. **Player queue vs. DJ queue (finding 1).** Read the count of `every track of player queue`, and separately confirm that the DJ queue is process-local by running `mu-go queue add "a"` and `mu-go queue` as two processes and recording both outputs.
4. **Stale install (finding 4).** Run `mu version` and `/opt/homebrew/bin/mu version` and record both.
5. **Playlist command shape (finding 5).** Run `mu-go playlist "Drill"` and `mu-go playlist play "Drill"` and record both exit codes.
6. **Boundary values (Review Focus #5).** Run the oracle with `volume 101`, `volume -1`, `volume abc`, `volume 0`, and `volume 100`, recording exit code and stderr for each. The question is whether out-of-range input is rejected with a clear message or silently clamped/wrapped. Note the expected fix lands in Plan 2, which owns command validation.

- [ ] **Step 4: Run the audit**

Run: `bun run audit`
Expected: exits 0 and writes `audit-out/audit.json`. Playback will start, skip, and change shuffle/repeat/love state. Leave playback paused afterward.

- [ ] **Step 5: Write the audit report**

`docs/superpowers/audit/2026-10-04-command-audit.md` with these sections:

- **Environment** — macOS version, Music.app presence, whether Automation permission is granted, Go oracle build hash.
- **Command matrix results** — a table: `command | invocation | exit | stdout (truncated) | stderr | verdict`, where verdict is exactly one of `ok`, `wrong output`, `leaks applescript`, `unusable`, `not audited (blocking)`.
- **Hypothesis findings** — for each of the six probes: hypothesis, observed evidence, confirmed or refuted, and the concrete repair.
- **Repair backlog** — one row per confirmed defect: `id | command | defect | proposed fix | spec §6 item`. This table is Plan 2's task list, so every row must name a specific file and function.

Rules for the report: no claim of "confirmed" without pasted observed output; every "ok" verdict must include the actual stdout; anything not exercised is marked `not audited` rather than assumed working.

- [ ] **Step 6: Verify the report has no unfilled cells**

Run: `grep -nE 'TBD|TODO|probably|assumed' docs/superpowers/audit/2026-10-04-command-audit.md`
Expected: no output. Every cell carries observed evidence.

- [ ] **Step 7: Commit**

```bash
git add scripts/audit.ts docs/superpowers/audit/2026-10-04-command-audit.md
git commit -m "docs: live command audit and repair backlog"
```

---

## Exit Criteria for Plan 1

- `bun test` passes; `bun run typecheck` exits 0.
- `src/applescript/` is the only place in the repo that spawns `osascript`.
- `docs/superpowers/audit/2026-10-04-command-audit.md` exists, has no unfilled cells, and its Repair Backlog names a file and function per row.
- The five Review Focus conditions each have a passing test.