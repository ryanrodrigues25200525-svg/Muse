/** ASCII Unit Separator — matches Go's `recordSep` in internal/music/controller.go. */
export const RECORD_SEP = "\u001f"

/**
 * Split a single delimited record into its fields. AppleScript builds these
 * with `ASCII character 31` joins, which is why track names containing commas,
 * quotes, or emoji survive intact.
 *
 * Field contents are returned verbatim: leading and trailing spaces inside a
 * title are meaningful and must not be trimmed.
 */
export function parseRecord(output: string): string[] {
  return output.split(RECORD_SEP)
}

/**
 * Split newline-separated rows into fields, using the legacy `|` separator.
 *
 * LEGACY FORMAT — CONSTRAINT: a field cannot contain `|` or a newline,
 * because both are structural here. Any AppleScript emitting this format must
 * escape or strip them, or the row will parse into extra fields.
 *
 * New AppleScript must emit RECORD_SEP-joined single records and be read with
 * `parseRecord`, which has no such limitation. This mirrors Go's
 * `parseTrackLines`, kept only so existing pipe-format callers have a parser.
 */
export function parseRows(output: string, sep = "|"): string[][] {
  return output
    .split("\n")
    .filter((line) => line.trim() !== "")
    .map((line) => line.split(sep))
}