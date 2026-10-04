import { AppleScriptError } from "./errors"

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
        const trimmed = stderr.trim()
        throw new AppleScriptError(trimmed, trimmed)
      }
      return stdout.trim()
    },
  }
}