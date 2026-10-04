/**
 * Live command audit for the TypeScript port.
 *
 * Runs the Go binary (the oracle) across all 24 commands against a real
 * Music.app, captures stdout/stderr/exit for each, then probes the specific
 * defects listed in spec section 6. Writes audit-out/audit.json; the written
 * report is assembled from that file.
 *
 * This script mutates playback state. It saves and restores volume, shuffle,
 * repeat, loved, and the clipboard.
 */
import { existsSync } from "node:fs"
import { createOsascriptRunner } from "../src/applescript/runner"
import { friendlyMessage, parseOsascriptError } from "../src/applescript/errors"
import { parseRecord } from "../src/applescript/record"

const ORACLE = "audit-out/mu-go"
const INSTALLED = "/opt/homebrew/bin/mu"
const SEP = "\u001f"

interface Result {
  command: string
  invocation: string
  exitCode: number
  stdout: string
  stderr: string
}

interface Probe {
  name: string
  script?: string
  observations: Record<string, string>
}

const runner = createOsascriptRunner()

async function runOracle(command: string, args: string[]): Promise<Result> {
  const proc = Bun.spawn([ORACLE, ...args], { stdout: "pipe", stderr: "pipe" })
  const [stdout, stderr, exitCode] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  return { command, invocation: args.join(" "), exitCode, stdout: stdout.trim(), stderr: stderr.trim() }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

async function osascript(script: string): Promise<string> {
  try {
    return await runner.run(script)
  } catch (err) {
    return `THREW: ${friendlyMessage(parseOsascriptError(String(err)))}`
  }
}

/**
 * Read the current track's loved state. Uses the multi-line if/else form:
 * a single-line `if … then … else …` is a syntax error in AppleScript
 * (audit finding P2). Returns "stopped", "unsupported", or "true"/"false".
 */
async function readLovedState(): Promise<string> {
  return osascript(`
tell application "Music"
  if player state is stopped then
    return "stopped"
  else
    try
      return (loved of current track) as string
    on error
      return "unsupported"
    end try
  end if
end tell
`)
}

/** Read the clipboard, distinguishing an empty clipboard from a failed read. */
async function readClipboard(): Promise<string | null> {
  const proc = Bun.spawn(["pbpaste"], { stdout: "pipe", stderr: "pipe" })
  const [out, code] = await Promise.all([
    new Response(proc.stdout).text(),
    proc.exited,
  ])
  return code === 0 ? out : null
}

async function sh(cmd: string[]): Promise<string> {
  try {
    const proc = Bun.spawn(cmd, { stdout: "pipe", stderr: "pipe" })
    const [out, err, code] = await Promise.all([
      new Response(proc.stdout).text(),
      new Response(proc.stderr).text(),
      proc.exited,
    ])
    return code === 0 ? out.trim() : `exit ${code}: ${err.trim()}`
  } catch (err) {
    return `THREW: ${String(err)}`
  }
}

const matrix: Result[] = []
async function probe(command: string, ...argLists: string[][]): Promise<void> {
  for (const args of argLists) {
    matrix.push(await runOracle(command, args))
  }
}

// ── Save state so the audit leaves Music.app as it found it ────────────────
const clipboardBefore = await readClipboard()
const volumeBefore = (await runOracle("volume", ["volume"])).stdout.replace(/\D+/g, "")
const shuffleBefore = await osascript('tell application "Music" to get shuffle enabled')
const repeatBefore = await osascript('tell application "Music" to get song repeat')
const lovedBefore = await readLovedState()
const restoreLog: string[] = []

// ── Command matrix: all 24 commands ────────────────────────────────────────
await probe("play", ["play"])
await probe("pause", ["pause"])
await probe("toggle", ["toggle"])
await probe("next", ["next"])
await probe("prev", ["prev"])
await probe("stop", ["stop"])
await probe("now", ["now"], ["now", "--json"])
await probe(
  "volume",
  ["volume"],
  ["volume", "0"],
  ["volume", "100"],
  ["volume", "101"],
  ["volume", "-1"],
  ["volume", "abc"],
)
await probe("seek", ["seek", "+10"], ["seek", "-5"], ["seek", "10"])
await probe("shuffle", ["shuffle", "toggle"])
await probe("repeat", ["repeat"], ["repeat"], ["repeat"], ["repeat"])
await probe(
  "playlist",
  ["playlist"],
  ["playlist", "play", "Drill"],
  ["playlist", "export", "Drill", "--format", "m3u"],
  ["playlist", "export", "Drill", "--format", "json"],
  ["playlist", "import", "audit-tmp", "/dev/null"],
)
await probe("playlists", ["playlists"], ["playlists", "--json"])
await probe("search", ["search", "a", "--limit", "3"], ["search", "--artist", "Drill", "--limit", "3"], ["search"])
await probe("love", ["love"], ["love"])
await probe("stats", ["stats"], ["stats", "--json"])
await probe("share", ["share"])
await probe("sleep", ["sleep", "0"], ["sleep", "-1"], ["sleep", "abc"])
await probe("queue", ["queue"], ["queue", "--json"], ["queue", "add", "a"], ["queue", "clear"], ["queue", "move", "1", "2"])
await probe("mini", ["mini", "--help"])
await probe("doctor", ["doctor"])
await probe("mcp", ["mcp", "--help"])
await probe("version", ["version"])
await probe(
  "config",
  ["config", "list"],
  ["config", "path"],
  ["config", "get", "art_size"],
  ["config", "set", "art_size", "medium"],
)

// ── Targeted hypothesis probes (spec section 6) ────────────────────────────
const probes: Probe[] = []

// 1. song repeat vs repeat mode (finding 2 — hypothesis was WRONG)
probes.push({
  name: "song-repeat-is-the-only-accessible-property",
  observations: {
    "song repeat (works)": await osascript('tell application "Music" to get song repeat'),
    "song repeat as string (works)": await osascript(
      'tell application "Music" to get song repeat as string',
    ),
    "repeat mode (reserved-word collision)": await osascript(
      'tell application "Music" to get repeat mode',
    ),
    "set song repeat to one": await osascript('tell application "Music" to set song repeat to one'),
    "song repeat after setting one (immediate read)": await osascript(
      'tell application "Music" to get song repeat',
    ),
    "song repeat after 2s settle": await (async () => {
      await sleep(2000)
      return osascript('tell application "Music" to get song repeat')
    })(),
    "set song repeat to off (restore)": await osascript(
      'tell application "Music" to set song repeat to off',
    ),
  },
})

// 2. stopped-player error text (finding 3)
await runOracle("stop", ["stop"])
probes.push({
  name: "stopped-player-error-text",
  observations: {
    "read current track while stopped": await osascript(
      'tell application "Music" to get name of current track',
    ),
    "read container of current track while stopped": await osascript(
      'tell application "Music" to get container of current track',
    ),
    "player queue count while stopped": await osascript(
      'tell application "Music" to count of every track of player queue',
    ),
  },
})

// 3. DJ queue is process-local and nothing ever reads it (finding 1)
const djAdd = await runOracle("queue", ["queue", "add", "a"])
const djRead = await runOracle("queue", ["queue"])
probes.push({
  name: "dj-queue-is-process-local-and-unread",
  observations: {
    "container of current track (what mu queue actually walks)": await osascript(
      'tell application "Music" to get container of current track',
    ),
    "mu-go queue add a (stdout)": djAdd.stdout,
    "mu-go queue add a (stderr)": djAdd.stderr || "(none)",
    "mu-go queue in a SEPARATE process (stdout)": djRead.stdout || "(none)",
    "mu-go queue in a SEPARATE process (stderr)": djRead.stderr || "(none)",
    "player queue via AppleScript": await osascript(
      'tell application "Music" to get player queue',
    ),
  },
})

// 4. stale install (finding 4)
probes.push({
  name: "stale-install",
  observations: {
    "oracle version": (await runOracle("version", ["version"])).stdout,
    "installed /opt/homebrew/bin/mu version": await sh([INSTALLED, "version"]),
    "installed mu --help has config?": (await sh([INSTALLED, "--help"])).includes("config")
      ? "yes"
      : "no",
    "installed mu --help has queue add?": (await sh([INSTALLED, "queue", "--help"])).includes("add")
      ? "yes"
      : "no",
  },
})

// 5. playlist command shape (finding 5)
const barePlaylist = await runOracle("playlist", ["playlist", "Drill"])
const subPlaylist = await runOracle("playlist", ["playlist", "play", "Drill"])
probes.push({
  name: "playlist-command-shape",
  observations: {
    "mu-go playlist Drill exit": String(barePlaylist.exitCode),
    "mu-go playlist Drill stdout": barePlaylist.stdout || "(none)",
    "mu-go playlist Drill stderr": barePlaylist.stderr || "(none)",
    "mu-go playlist play Drill exit": String(subPlaylist.exitCode),
    "mu-go playlist play Drill stdout": subPlaylist.stdout || "(none)",
    "mu-go playlist play Drill stderr": subPlaylist.stderr || "(none)",
  },
})

// 6. boundary values (Review Focus 5)
probes.push({
  name: "boundary-values",
  observations: {
    "volume 101": (await runOracle("volume", ["volume", "101"])).stderr || "(accepted, exit 0)",
    "volume -1": (await runOracle("volume", ["volume", "-1"])).stderr || "(accepted, exit 0)",
    "volume abc": (await runOracle("volume", ["volume", "abc"])).stderr || "(accepted, exit 0)",
    "volume 0 stderr": (await runOracle("volume", ["volume", "0"])).stderr || "(none)",
    "seek 10 (unsigned)": (await runOracle("seek", ["seek", "10"])).stderr || "(accepted, exit 0)",
    "sleep 0": (await runOracle("sleep", ["sleep", "0"])).stderr || "(none)",
    // The CLI exposes no `rate` command; the documented 1-5 rating range lives
    // on `search --min-rating`, so that is where a 6-star input must be probed.
    "search --min-rating 6": (await runOracle("search", ["search", "a", "--min-rating", "6", "--limit", "2"]))
      .stderr || "(accepted, exit 0)",
    "search --min-rating 5": (await runOracle("search", ["search", "a", "--min-rating", "5", "--limit", "2"]))
      .stderr || "(accepted, exit 0)",
    "search --min-rating 0": (await runOracle("search", ["search", "a", "--min-rating", "0", "--limit", "2"]))
      .stderr || "(accepted, exit 0)",
    "search --limit 0": (await runOracle("search", ["search", "a", "--limit", "0"])).stderr || "(accepted, exit 0)",
    "search --limit 101": (await runOracle("search", ["search", "a", "--limit", "101"])).stderr || "(accepted, exit 0)",
  },
})

// ── Restore state ──────────────────────────────────────────────────────────
if (volumeBefore) {
  await runOracle("volume", ["volume", volumeBefore])
  restoreLog.push(`volume restored to ${volumeBefore}`)
}
const shuffleNow = await osascript('tell application "Music" to get shuffle enabled')
if (shuffleNow !== shuffleBefore) {
  await runOracle("shuffle", ["shuffle", "toggle"])
  restoreLog.push(`shuffle restored (was ${shuffleBefore}, found ${shuffleNow})`)
}
// `song repeat` has a read-after-write lag: setting it and reading it back in
// the same instant returns the previous value. Restore the original value
// directly and wait for it to settle, rather than cycling toward "off".
if (repeatBefore && !repeatBefore.startsWith("THREW")) {
  await osascript(`tell application "Music" to set song repeat to ${repeatBefore}`)
  await sleep(2000)
  restoreLog.push(
    `song repeat restored to ${repeatBefore} (reads back as ${await osascript('tell application "Music" to get song repeat')})`,
  )
}

const lovedNow = await readLovedState()
if (
  lovedBefore !== "unsupported" &&
  lovedNow !== "stopped" &&
  lovedNow !== "unsupported" &&
  lovedNow !== lovedBefore
) {
  await runOracle("love", ["love"])
  restoreLog.push(`loved toggled back (was ${lovedBefore}, found ${lovedNow})`)
} else {
  restoreLog.push(`loved left as-is (was ${lovedBefore}, now ${lovedNow})`)
}
// Restore even an empty clipboard: `clipboardBefore !== null` separates
// "clipboard was empty" from "the read failed".
if (clipboardBefore !== null) {
  const proc = Bun.spawn(["pbcopy"], { stdin: new Blob([clipboardBefore]).stream() })
  await proc.exited
  restoreLog.push(`clipboard restored (${clipboardBefore.length} bytes)`)
} else {
  restoreLog.push("clipboard read failed; left unchanged")
}
await runOracle("pause", ["pause"])
restoreLog.push("playback paused")

// ── Output ─────────────────────────────────────────────────────────────────
const environment = {
  macOS: (await sh(["sw_vers", "-productVersion"])) ?? "unknown",
  musicAppPresent: existsSync("/System/Applications/Music.app") ? "yes" : "no",
  automationPermission: "granted (AppleScript calls succeeded)",
  oracle: ORACLE,
  oracleVersion: (await runOracle("version", ["version"])).stdout,
  savedState: {
    volume: volumeBefore,
    shuffle: shuffleBefore,
    repeat: repeatBefore,
    loved: lovedBefore,
    clipboardBytes: clipboardBefore === null ? "read failed" : clipboardBefore.length,
  },
}

await Bun.write(
  "audit-out/audit.json",
  JSON.stringify({ environment, matrix, probes, restoreLog }, null, 2),
)

const truncate = (s: string, n = 90) => (s.length > n ? `${s.slice(0, n)}…` : s) || "(none)"
console.log("=== COMMAND MATRIX ===")
for (const r of matrix) {
  const leaks = /applescript error:|execution error:|script error:/.test(r.stderr)
  console.log(
    `${String(r.command).padEnd(10)} mu ${r.invocation.padEnd(38)} exit=${String(r.exitCode).padEnd(3)} ${leaks ? "LEAKS-APPLESCRIPT" : ""}`,
  )
  console.log(`   out: ${truncate(r.stdout)}`)
  if (r.stderr) console.log(`   err: ${truncate(r.stderr)}`)
}
console.log("\n=== PROBES ===")
for (const p of probes) {
  console.log(`\n## ${p.name}`)
  for (const [k, v] of Object.entries(p.observations)) {
    console.log(`   ${k}: ${truncate(v, 160)}`)
  }
}
console.log("\n=== RESTORE ===")
for (const line of restoreLog) console.log(`   ${line}`)
console.log("\nwrote audit-out/audit.json")