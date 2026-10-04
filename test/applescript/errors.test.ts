import { expect, test } from "bun:test"
import {
  AppleScriptError,
  friendlyMessage,
  parseOsascriptError,
} from "../../src/applescript/errors"

const NO_CURRENT_TRACK =
  "execution error: Music got an error: Can't get container of current track. (-1728)"
const NOT_AUTHORIZED =
  "execution error: Not authorized to send Apple events to Music. (-1743)"

test("parseOsascriptError extracts the osascript error number", () => {
  expect(parseOsascriptError(NO_CURRENT_TRACK).osascriptNumber).toBe(1728)
  expect(parseOsascriptError(NOT_AUTHORIZED).osascriptNumber).toBe(1743)
  expect(parseOsascriptError("execution error: whatever (-1234)").osascriptNumber).toBe(1234)
})

test("parseOsascriptError returns null when there is no error number", () => {
  expect(parseOsascriptError("command not found").osascriptNumber).toBeNull()
})

test("parseOsascriptError keeps the raw stderr for diagnostics", () => {
  expect(parseOsascriptError(NO_CURRENT_TRACK).rawOutput).toBe(NO_CURRENT_TRACK)
})

test("friendlyMessage explains a stopped player instead of leaking AppleScript", () => {
  expect(friendlyMessage(parseOsascriptError(NO_CURRENT_TRACK))).toBe(
    "Nothing is playing. Start playback with: mu play",
  )
})

test("friendlyMessage points at Automation settings when permission is denied", () => {
  expect(friendlyMessage(parseOsascriptError(NOT_AUTHORIZED))).toBe(
    "Not authorized to send Apple events to Music. Open System Settings → Privacy & Security → Automation and grant your terminal access to Music.app",
  )
})

test("friendlyMessage keeps an unrecognized error's own text", () => {
  const message = friendlyMessage(parseOsascriptError("execution error: whatever (-1234)"))
  expect(message).toContain("whatever")
  expect(message).not.toContain("execution error:")
})

test("friendlyMessage handles output with no error number", () => {
  expect(friendlyMessage(parseOsascriptError("command not found"))).toContain(
    "command not found",
  )
})

test("friendlyMessage only blames a stopped player for a 1728 about the current track", () => {
  // -1728 is a generic AppleScript error. A playlist lookup failing with the
  // same number must not be reported as "nothing is playing".
  const unrelated = parseOsascriptError(
    "execution error: Can't get playlist \"Nope\". (-1728)",
  )
  expect(friendlyMessage(unrelated)).not.toContain("Nothing is playing")
  expect(friendlyMessage(unrelated)).toContain("Nope")
})

test("friendlyMessage passes through a plain Error", () => {
  expect(friendlyMessage(new Error("plain"))).toBe("plain")
})

test("friendlyMessage never emits the osascript wrapper noise", () => {
  const err = new AppleScriptError("wrapped", "wrapped")
  expect(friendlyMessage(err)).not.toContain("osascript error:")
})

test("friendlyMessage stringifies a thrown non-Error value", () => {
  expect(friendlyMessage("just a string")).toBe("just a string")
})