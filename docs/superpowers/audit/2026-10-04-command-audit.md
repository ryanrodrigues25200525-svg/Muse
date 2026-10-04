# Muse Command Audit — 2026-10-04

Live audit of all 24 commands against a real Music.app, run from the Go oracle
binary at commit `de57c60`. This report is Plan 2's task list: every row in the
Repair Backlog names a file and function to change.

Raw evidence: `audit-out/audit.json`, produced by `bun run audit` (`scripts/audit.ts`).

## Environment

| Item | Value |
| --- | --- |
| macOS | 27.0.1 (26A434) |
| Music.app | present at `/System/Applications/Music.app`, running |
| Automation permission | granted — every AppleScript call in this audit succeeded |
| Go oracle | `audit-out/mu-go`, built from `de57c60`, reports `Muse dev` |
| Installed binary | `/opt/homebrew/bin/mu`, reports `Muse dev` |
| State saved before audit | volume 100, shuffle false, song repeat `all`, playback paused |
| State after audit + manual correction | volume 100, shuffle false, song repeat `all`, playback paused |

### Side effects during the audit

Playback started, tracks skipped, and shuffle/repeat/love were toggled, as approved.
`mu share` overwrote the clipboard; the original clipboard content was restored.

One correction to note: the audit script's first version restored `song repeat` by
cycling toward `off` instead of back to its recorded original value (`all`), leaving
repeat set to `off`. That was detected, fixed in `scripts/audit.ts`, and the machine
was returned to `song repeat = all` by hand. The script now restores the recorded
value directly and waits for it to settle.

## Command matrix results

| command | invocation | exit | stdout (truncated) | stderr | verdict |
| --- | --- | --- | --- | --- | --- |
| play | `play` | 0 | `▶ Playing` | | ok |
| pause | `pause` | 0 | `⏸ Paused` | | ok |
| toggle | `toggle` | 0 | `⏯ Toggled` | | ok |
| next | `next` | 0 | `⏭ Next track` | | ok |
| prev | `prev` | 0 | `⏮ Previous track` | | ok |
| stop | `stop` | 0 | `⏹ Stopped` | | ok |
| now | `now` | 0 | `⏹ Stopped` | | ok |
| now | `now --json` | 0 | `{"title": "", "state": "stopped", "volume": 100…}` | | ok |
| volume | `volume` | 0 | `♪  Volume: 100%` | | ok |
| volume | `volume 0` | 0 | `♪  Volume set to 0%` | | ok |
| volume | `volume 100` | 0 | `♪  Volume set to 100%` | | ok |
| volume | `volume 101` | 1 | | `Volume must be 0–100` | ok |
| volume | `volume -1` | 1 | `unknown shorthand flag: '1' in -1` | `Error: unknown shorthand flag: '1' in -1` | unusable |
| volume | `volume abc` | 1 | | `Volume must be 0–100` | ok |
| seek | `seek +10` | 1 | | `Error: applescript error: exit status 1 (output: 143:147: syntax error: Expected "end" or "end tell" but found "else". (-2741)` | unusable |
| seek | `seek -5` | 1 | `unknown shorthand flag: '5' in -5` | `Error: unknown shorthand flag: '5' in -5` | unusable |
| seek | `seek 10` | 1 | | `seek value must include a sign, for example +30 or -10` | ok |
| shuffle | `shuffle toggle` | 0 | `⇄ Shuffle off` | | ok |
| repeat | `repeat` | 0 | `↻ Repeat one` | | wrong output |
| repeat | `repeat` | 0 | `↻ Repeat off` | | ok |
| repeat | `repeat` | 0 | `↻ Repeat off` | | wrong output |
| repeat | `repeat` | 0 | `↻ Repeat all` | | ok |
| playlist | `playlist` | 0 | `Manage and play playlists` + usage | | wrong output |
| playlist | `playlist play Drill` | 0 | `Playing playlist: Drill...` | | ok |
| playlist | `playlist export Drill --format m3u` | 0 | `#EXTM3U` + `#EXTINF:0,Abra Cadabra - On Deck` | | ok |
| playlist | `playlist export Drill --format json` | 0 | `[{"title": "On Deck", "artist": "Abra Cadabra"…}]` | | ok |
| playlist | `playlist import audit-tmp /dev/null` | 1 | | `Error: cannot determine format; use a .m3u/.json file or pass raw content with detectable format` | ok |
| playlists | `playlists` | 0 | `My Playlists: Library (196 tracks)…` | | ok |
| playlists | `playlists --json` | 0 | `{"mine": [{"name": "Library", "id": 62…}]}` | | ok |
| search | `search a --limit 3` | 0 | `3 result(s) for "a": [1] Two Aladins…` | | ok |
| search | `search --artist Drill --limit 3` | 0 | `No tracks found.` | | ok |
| search | `search` | 1 | | `Error: provide a search query or at least one filter (--artist, --album…)` | ok |
| love | `love` | 1 | | `Could not toggle loved: loved is not supported for this track` | ok |
| love | `love` | 1 | | `Could not toggle loved: loved is not supported for this track` | ok |
| stats | `stats` | 0 | `🎵 Say Less 👤 Meekz 💿 Say Less - Single` | | ok |
| stats | `stats --json` | 0 | `{"title": "Say Less", "artist": "Meekz"…}` | | ok |
| share | `share` | 0 | `Copied to clipboard: Say Less — Meekz` | | ok |
| sleep | `sleep 0` | 1 | | `Please provide a positive number of minutes.` | ok |
| sleep | `sleep -1` | 1 | `unknown shorthand flag: '1' in -1` | `Error: unknown shorthand flag: '1' in -1` | unusable |
| sleep | `sleep abc` | 1 | | `Please provide a positive number of minutes.` | ok |
| queue | `queue` | 0 | `Up Next: [1] Don't Like Drill — Meekz & Central Cee…` | | ok |
| queue | `queue --json` | 0 | `[{"title": "Don't Like Drill", "artist": "Meekz & Central Cee"…}]` | | ok |
| queue | `queue add a` | 0 | `Queued: Two Aladins (Libertad Vocal Mix) [Mixed] – Aaron Sevilla… (1 in queue)` | | unusable |
| queue | `queue clear` | 0 | `DJ queue cleared.` | | wrong output |
| queue | `queue move 1 2` | 1 | | `Error: from index 1 out of range (1-0)` | unusable |
| mini | `mini --help` | 0 | `Show a compact single-line now-playing display…` | | not audited (blocking) |
| doctor | `doctor` | 0 | `macOS detected ✅ osascript available ✅…` | | ok |
| mcp | `mcp --help` | 0 | `Run the Muse MCP server over stdio` | | not audited (blocking) |
| version | `version` | 0 | `Muse dev` | | ok |
| config | `config list` | 0 | `art_size = medium  default_repeat = off…` | | ok |
| config | `config path` | 0 | `/Users/ryanrodrigues/.config/muse/config.json` | | ok |
| config | `config get art_size` | 0 | `medium` | | ok |
| config | `config set art_size medium` | 0 | `Set art_size = medium` | | ok |

`mu sleep` beyond argument validation is also `not audited (blocking)` — a positive
value blocks for its full duration. Plan 2 audits it under a PTY.

## Hypothesis findings

### H1 — `seek` has never worked (CONFIRMED, spec §6 had no entry for this)

Hypothesis: the README's `mu seek +30` may fail. **Confirmed** — it fails with an
AppleScript syntax error, so the documented command has never worked:

```
$ mu-go seek +10
Error: applescript error: exit status 1 (output: 143:147: syntax error:
Expected "end" or "end tell" but found "else". (-2741))
```

Root cause is in `internal/music/controller.go:320` `Seek`. It emits

```applescript
if newPos < 0 then set player position to 0
else set player position to newPos
```

AppleScript's single-line `if … then …` form cannot take an `else` clause on a
following line. The verified fix returns the new position:

```applescript
if newPos < 0 then
    set player position to 0
else
    set player position to newPos
end if
```

### H2 — `repeat` reads the wrong property (REFUTED as stated, but a real bug remains)

The hypothesis was that `CycleRepeat` compares `song repeat as string` against
`"off"`/`"all"`/`"one"` and therefore never matches. **Refuted.** `song repeat` is
the correct property, and `song repeat as string` does return `off`/`all`/`one`:

```
song repeat (works):                 all
song repeat as string (works):       all
repeat mode (reserved-word collision): syntax error: Expected expression but found "repeat". (-2741)
```

`repeat mode` is unreachable because `repeat` is an AppleScript reserved word — the
Go code is right to avoid it.

A different defect is real. `song repeat` has a **read-after-write lag**: setting and
reading it in the same AppleScript invocation returns the previous value.

```
$ osascript -e 'tell application "Music" to set song repeat to off' \
             -e 'tell application "Music" to get song repeat'
all            <- stale: the write has not landed yet
$ sleep 3; osascript -e 'tell application "Music" to get song repeat'
off            <- settled
```

`CycleRepeat` reads that stale value, so rapid calls do not advance the cycle:

```
call 1: ↻ Repeat one
call 2: ↻ Repeat one     <- duplicate: acted on stale state
call 3: ↻ Repeat off
call 4: ↻ Repeat all
```

The same duplicate appeared in the command matrix (`one`, `off`, `off`, `all`).

### H3 — raw AppleScript text leaks to users (CONFIRMED)

`RunAppleScript` returns osascript's combined output verbatim
(`internal/music/controller.go:65`), so users see:

```
Error: applescript: Music got an error: Can't get container of current track.
```

Reproduced by `mu queue` while stopped. `friendlyMessage` from
`src/applescript/errors.ts` already maps this case correctly — probed live:

```
read current track while stopped: Nothing is playing. Start playback with: mu play
read container of current track while stopped: Nothing is playing. Start playback with: mu play
```

The port's job is to route every command's errors through it.

### H4 — the DJ queue is dead code (CONFIRMED, sharper than the hypothesis)

The hypothesis was that `mu queue` and `mu queue add` manage different lists.
Evidence shows something worse: **nothing can ever read what `queue add` writes.**

```
mu-go queue add a (stdout): Queued: Two Aladins (Libertad Vocal Mix) [Mixed] – Aaron Sevilla… (1 in queue)
mu-go queue in a SEPARATE process (stdout): (none)
mu-go queue in a SEPARATE process (stderr): Error: applescript: Music got an error: Can't get container of current track.
```

The entry is appended to a package-level Go slice (`internal/music/queue.go:16`) and
dies with the process. Its only readers (`DJGetQueue`, `DJQueueLen`, `DJDequeueNext`,
`DJQueueCopy`) are called from the TUI, which is equally short-lived.

Meanwhile `mu queue` never touches Music's player queue at all.
`internal/music/controller.go:681` `GetQueue` **synthesizes** "Up Next" by walking
forward through the current track's container playlist and wrapping around, emitting
up to 10 tracks. Music's actual player queue is not addressable through the
AppleScript forms tried:

```
player queue via AppleScript: syntax error: A identifier can't go after this identifier. (-2740)
count of every track of player queue: syntax error (-2741)
count of (every track of player queue): syntax error (-2741)
count tracks of player queue: syntax error (-2740)
```

So `queue add` reports success for an action with no reachable effect, `queue move`
can only fail (`from index 1 out of range (1-0)`, since a fresh process has an empty
queue), and `queue clear` clears nothing.

### H5 — the installed binary is stale (CONFIRMED)

```
installed /opt/homebrew/bin/mu version: Muse dev
installed mu --help has config?: no
installed mu --help has queue add?: no
```

The binary on `PATH` predates the `config` command and the `queue` subcommands, so
`mu config list` and `mu queue add` fail for anyone using the installed copy while
the README documents both.

### H6 — `playlist` silently no-ops (CONFIRMED)

README documents `mu playlist "Gym"`. The command group requires a subcommand:

```
mu-go playlist Drill exit: 0
mu-go playlist Drill stdout: Manage and play playlists\n\nUsage:\n  mu playlist [command]…
mu-go playlist play Drill exit: 0
mu-go playlist play Drill stdout: Playing playlist: Drill...
```

`mu playlist "Drill"` prints help and exits 0. A user following the README gets no
error and no playback.

### H7 — boundary validation (MOSTLY CORRECT)

| Input | Result | Assessment |
| --- | --- | --- |
| `volume 101` | `Volume must be 0–100`, exit 1 | correct |
| `volume abc` | `Volume must be 0–100`, exit 1 | correct |
| `volume 0` / `volume 100` | accepted | correct |
| `volume -1` | `unknown shorthand flag: '1' in -1` | wrong message, see R2 |
| `seek 10` (unsigned) | `seek value must include a sign…` | correct |
| `sleep 0` / `sleep abc` | `Please provide a positive number of minutes.` | correct |
| `playlist import … /dev/null` | clean rejection, no playlist created | correct |

The defect is not validation logic but argument parsing: any argument beginning with
`-` followed by a digit is consumed as a flag by pflag, so `volume -1`, `seek -5`, and
`sleep -1` never reach the validator. `mu seek -10` is in the README.

### H8 — `love` on unsupported tracks (LIMITATION, not a defect)

```
$ mu-go love
Could not toggle loved: loved is not supported for this track
```

Both audited tracks were streamed/Apple Music titles where `loved` is genuinely
unsupported. The message is accurate and exits 1. Recorded as a documented platform
limitation rather than a repair.

## Repair Backlog

Plan 2's task list. TypeScript paths are the target; the Go file is cited as the
origin of the defect.

| id | command | defect | proposed fix | origin | spec §6 |
| --- | --- | --- | --- | --- | --- |
| R1 | `seek` | AppleScript syntax error — single-line `if … then` cannot take `else` on the next line, so `mu seek +30` has never worked | `src/music/playback.ts` `seek()`: emit the multi-line `if/then/else/end if` form proven in H1; add a test asserting the emitted script contains `end if` | `controller.go:320` `Seek` | new |
| R2 | `seek`, `volume`, `sleep` | arguments like `-5`/`-1` are parsed as flags, so negative values never reach validation and `mu seek -10` from the README fails | CLI arg handling: treat a leading `-` followed by a digit as a positional; keep `--` working; assert `mu seek -5` reaches the range validator | pflag behaviour in `commands.go` | new |
| R3 | `queue add`, `queue clear`, `queue move` | entries go to a process-local slice nothing reads; `add` reports success for an unreachable effect, `move` can only fail | Decide with the user: delete the three subcommands, or persist the queue and have `mu queue` read it. Do not port the current behavior | `queue.go:16` | item 1 |
| R4 | all | raw osascript output reaches the user, e.g. `Error: applescript: Music got an error: Can't get container of current track.` | route every command error through `friendlyMessage` from `src/applescript/errors.ts`; add a test per command that no command prints `applescript error:` | `controller.go:65` `RunAppleScript` | item 3 |
| R5 | `playlist` | `mu playlist "X"` prints help and exits 0 instead of playing | accept a bare name as an alias for `playlist play <name>`, or error with a pointer to the subcommand; align README either way | `commands.go:312` | item 5 |
| R6 | install | `/opt/homebrew/bin/mu` predates `config` and the `queue` subcommands, so documented commands are missing | rebuild and reinstall; add an installed-version line to `doctor` so drift is visible | packaging | item 4 |
| R7 | `repeat` | read-after-write lag on `song repeat` makes cycling unreliable and can repeat a state | after writing, poll `song repeat` until it reflects the intended value or a timeout elapses; never use `repeat mode` (reserved word) | `controller.go:299` `CycleRepeat` | item 2, revised |
| R8 | `love` | fails on tracks where `loved` is unsupported | keep the accurate message; document the limitation in README | `controller.go:433` | new |
| R9 | `sleep` | help text and README never state that the argument is minutes | document the unit in the command help and README | `commands.go:645` | item 6 |
| R10 | `lyrics`, `art` | implemented in Go, unreachable from the CLI | expose `mu lyrics` and `mu art` so the ported code is not dead | `internal/lyrics`, `internal/art` | item 7 |

## Porting constraints discovered

Facts the TypeScript port must respect. Each was verified live.

| id | constraint | evidence |
| --- | --- | --- |
| P1 | Never use `repeat mode` — `repeat` is an AppleScript reserved word and the property is unaddressable. Use `song repeat`. | H2 |
| P2 | A single-line `if … then …` cannot be followed by `else`. Always use the multi-line form with `end if`. | H1 |
| P3 | `song repeat` has a ~1–3s read-after-write lag; never read it in the same invocation that writes it. | H2 |
| P4 | Music's player queue is not addressable through the AppleScript forms tried, so "Up Next" must keep being synthesized from the current track's container playlist. | H4 |
| P5 | Column padding must not assume one character per column. The live library contains `🇦🇪`, `العربية`, `Sharma (ਸ਼ਰਮਾ) Music`, and `🔥🔥❤️`, all of which break `%-32s`-style alignment. | `playlists` output |
| P6 | `friendlyMessage`'s `-1728` → "Nothing is playing" mapping is confirmed correct against real osascript output, and the guard against unrelated `-1728` errors is required. | H3 |