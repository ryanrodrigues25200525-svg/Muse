import { expect, test } from "bun:test"
import { MUSE_VERSION } from "../src/version"

test("MUSE_VERSION is a non-empty string", () => {
  expect(typeof MUSE_VERSION).toBe("string")
  expect(MUSE_VERSION.length).toBeGreaterThan(0)
})

test("MUSE_VERSION defaults to dev when not injected at build time", () => {
  // Under bun test nothing defines MUSE_VERSION, so the default must survive.
  expect(MUSE_VERSION).toBe("dev")
})