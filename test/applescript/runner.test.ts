import { describe, expect, test } from "bun:test"
import { AppleScriptError, friendlyMessage } from "../../src/applescript/errors"
import { createOsascriptRunner } from "../../src/applescript/runner"

// Every test here shells out to the real osascript binary.
const describeMac = process.platform === "darwin" ? describe : describe.skip

describeMac("createOsascriptRunner", () => {
  test("returns an object exposing run", () => {
    const runner = createOsascriptRunner()
    expect(typeof runner.run).toBe("function")
  })

  test("resolves with the script's return value", async () => {
    const runner = createOsascriptRunner()
    expect(await runner.run('return "ok"')).toBe("ok")
  })

  test("trims surrounding whitespace from stdout", async () => {
    const runner = createOsascriptRunner()
    expect(await runner.run('return "  padded  "')).toBe("padded")
  })

  test("rejects with AppleScriptError carrying raw stderr", async () => {
    const runner = createOsascriptRunner()
    let caught: unknown
    try {
      await runner.run('error "boom" number -1728')
      throw new Error("expected run to reject, but it resolved")
    } catch (err) {
      caught = err
    }
    expect(caught).toBeInstanceOf(AppleScriptError)
    expect((caught as AppleScriptError).rawOutput).toContain("boom")
  })

  test("runner errors carry a parsed number, so friendlyMessage can dispatch", async () => {
    // The transport is the only place real osascript failures enter the
    // system. If it does not parse the number, every command downstream
    // falls back to printing raw AppleScript text.
    const runner = createOsascriptRunner()
    let caught: unknown
    try {
      await runner.run('error "Not authorized to send Apple events to Music." number -1743')
      throw new Error("expected run to reject, but it resolved")
    } catch (err) {
      caught = err
    }
    expect(caught).toBeInstanceOf(AppleScriptError)
    expect((caught as AppleScriptError).osascriptNumber).toBe(1743)
    expect(friendlyMessage(caught)).toContain("Privacy & Security")
  })
})