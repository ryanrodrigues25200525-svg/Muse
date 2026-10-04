import { expect, test } from "bun:test"
import { parseRecord, parseRows, RECORD_SEP } from "../../src/applescript/record"

test("RECORD_SEP is ASCII character 31", () => {
  expect(RECORD_SEP).toBe("\u001f")
})

test("parseRecord splits a single delimited record", () => {
  expect(parseRecord("a\u001fb\u001fc")).toEqual(["a", "b", "c"])
})

test("parseRecord on empty output yields one empty field", () => {
  expect(parseRecord("")).toEqual([""])
})

test("parseRecord on a single field yields one field", () => {
  expect(parseRecord("solo")).toEqual(["solo"])
})

test("parseRecord preserves non-ASCII field contents", () => {
  // A real library has emoji and non-Latin script in names; a naive split
  // must not corrupt or drop them.
  expect(parseRecord("Sharma (ਸ਼ਰਮਾ) Music\u001f🔥🔥❤️\u001fR&B")).toEqual([
    "Sharma (ਸ਼ਰਮਾ) Music",
    "🔥🔥❤️",
    "R&B",
  ])
})

test("parseRecord keeps newlines inside a field rather than flattening them", () => {
  expect(parseRecord("two\nlines\u001fb")).toEqual(["two\nlines", "b"])
})

test("parseRows splits newline-separated rows on the given separator", () => {
  expect(parseRows("a|b|c\nd|e|f", "|")).toEqual([
    ["a", "b", "c"],
    ["d", "e", "f"],
  ])
})

test("parseRows defaults to the pipe separator", () => {
  expect(parseRows("a|b|c")).toEqual([["a", "b", "c"]])
})

test("parseRows drops blank lines", () => {
  expect(parseRows("a|b|c\n\nd|e|f", "|")).toEqual([
    ["a", "b", "c"],
    ["d", "e", "f"],
  ])
})

test("parseRows preserves non-ASCII fields byte for byte", () => {
  expect(parseRows("🇦🇪|العربية|/ghazal\n🇬🇧|UK|rap", "|")).toEqual([
    ["🇦🇪", "العربية", "/ghazal"],
    ["🇬🇧", "UK", "rap"],
  ])
})

test("parseRows keeps significant spaces inside fields", () => {
  expect(parseRows("a| padded |c", "|")).toEqual([["a", " padded ", "c"]])
})