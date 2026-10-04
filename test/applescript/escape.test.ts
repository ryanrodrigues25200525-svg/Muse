import { expect, test } from "bun:test"
import { escapeAppleScriptString } from "../../src/applescript/escape"

test("passes plain text through unchanged", () => {
  expect(escapeAppleScriptString("plain")).toBe("plain")
})

test("escapes double quotes", () => {
  expect(escapeAppleScriptString('say "hi"')).toBe('say \\"hi\\"')
})

test("escapes backslashes", () => {
  expect(escapeAppleScriptString("back\\slash")).toBe("back\\\\slash")
})

test("flattens newlines to spaces", () => {
  expect(escapeAppleScriptString("two\nlines")).toBe("two lines")
})

test("flattens carriage returns to spaces", () => {
  expect(escapeAppleScriptString("two\rlines")).toBe("two lines")
})