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
 * Split newline-separated rows into fields. Blank lines are dropped so a
 * trailing newline in osascript output does not produce an empty row.
 *
 * The default separator matches Go's `parseTrackLines`, which delimits each
 * row's fields with `|`.
 */
export function parseRows(output: string, sep = "|"): string[][] {
  return output
    .split("\n")
    .filter((line) => line.trim() !== "")
    .map((line) => line.split(sep))
}