/** osascript reports `LINE:COL: execution error: <message> (<number>)`. */
const LEADING_NOISE = /^(?:\d+:\d+:\s*)?(?:(?:execution|script) error:\s*)?/

/** The trailing parenthesised osascript error number, on the last non-empty line. */
const TRAILING_NUMBER = /\((-?\d+)\)\s*$/

/**
 * A -1728 raised while Music has no current track. The number alone is not
 * enough: -1728 is a generic AppleScript error, so the message must also point
 * at the current track before we tell the user nothing is playing.
 */
const ABOUT_CURRENT_TRACK = /current track|container of/i

const FRIENDLY_NOTHING_PLAYING = "Nothing is playing. Start playback with: mu play"
const FRIENDLY_NOT_AUTHORIZED =
  "Not authorized to send Apple events to Music. Open System Settings → Privacy & Security → Automation and grant your terminal access to Music.app"

/**
 * A failure from the osascript process. Task 3 layers parsed error numbers
 * and user-facing message mapping on top of this.
 */
export class AppleScriptError extends Error {
  constructor(
    message: string,
    readonly rawOutput: string,
    readonly osascriptNumber: number | null = null,
  ) {
    super(message)
    this.name = "AppleScriptError"
  }
}

/** Strip osascript's `6:12: execution error: ` wrapper from a message. */
function stripLeadingNoise(text: string): string {
  return text.replace(LEADING_NOISE, "").trim()
}

/** Build an `AppleScriptError` from raw osascript stderr. */
export function parseOsascriptError(stderr: string): AppleScriptError {
  const trimmed = stderr.trim()
  const lines = trimmed.split("\n").filter((line) => line.trim() !== "")
  const lastLine = lines[lines.length - 1] ?? ""
  const match = TRAILING_NUMBER.exec(lastLine)
  // osascript prints signed numbers (-1743). Store the magnitude so dispatch
  // tables can key on positive values; rawOutput keeps the original text.
  const osascriptNumber = match ? Math.abs(Number.parseInt(match[1]!, 10)) : null

  return new AppleScriptError(stripLeadingNoise(trimmed), stderr, osascriptNumber)
}

/**
 * Turn any thrown value into a message safe to print. No caller should ever
 * surface osascript's raw output to a user.
 */
export function friendlyMessage(err: unknown): string {
  if (err instanceof AppleScriptError) {
    if (err.osascriptNumber === 1743) {
      return FRIENDLY_NOT_AUTHORIZED
    }
    if (err.osascriptNumber === 1728 && ABOUT_CURRENT_TRACK.test(err.message)) {
      return FRIENDLY_NOTHING_PLAYING
    }
    return err.message
  }
  return err instanceof Error ? err.message : String(err)
}