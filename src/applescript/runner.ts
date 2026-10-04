import { parseOsascriptError } from "./errors"

/**
 * The single seam through which Muse reaches Music.app. Every AppleScript
 * call takes a ScriptRunner so tests can assert emitted script text without
 * invoking osascript.
 */
export interface ScriptRunner {
  run(script: string): Promise<string>
}

export function createOsascriptRunner(): ScriptRunner {
  return {
    async run(script: string): Promise<string> {
      const proc = Bun.spawn(["osascript", "-e", script], {
        stdout: "pipe",
        stderr: "pipe",
      })
      const [stdout, stderr, exitCode] = await Promise.all([
        new Response(proc.stdout).text(),
        new Response(proc.stderr).text(),
        proc.exited,
      ])

      if (exitCode !== 0) {
        // Parse here, at the only place real osascript output enters the
        // system, so every downstream caller can dispatch on the error number
        // instead of printing raw AppleScript text.
        throw parseOsascriptError(stderr)
      }
      return stdout.trim()
    },
  }
}