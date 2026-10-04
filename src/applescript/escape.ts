/**
 * Escape a value for embedding inside an AppleScript string literal.
 *
 * Order matters: backslashes must be doubled first, or the backslashes
 * introduced by the quote and control-character escapes would be doubled
 * again. Newlines and carriage returns become spaces because an AppleScript
 * string literal cannot span lines safely.
 *
 * Mirrors Go's `escapeAS` in internal/music/controller.go.
 */
export function escapeAppleScriptString(value: string): string {
  return value
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\r/g, " ")
    .replace(/\n/g, " ")
}