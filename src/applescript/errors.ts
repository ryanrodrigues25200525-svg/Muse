/**
 * A failure from the osascript process. Task 3 layers parsed error numbers
 * and user-facing message mapping on top of this.
 */
export class AppleScriptError extends Error {
  constructor(
    message: string,
    readonly rawOutput: string,
  ) {
    super(message)
    this.name = "AppleScriptError"
  }
}