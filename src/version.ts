// Release builds replace this literal via:
//   bun build --compile --define MUSE_VERSION='"x.y.z"'
// The default must stay "dev" so the test suite sees an uninjected value.
export const MUSE_VERSION = "dev"