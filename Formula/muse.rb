class Muse < Formula
  desc "CLI, TUI, and MCP controller for Apple Music on macOS"
  homepage "https://github.com/ryanrodrigues25200525-svg/Muse"
  url "https://github.com/ryanrodrigues25200525-svg/Muse/releases/download/v0.2.0/muse_0.2.0_darwin_universal.tar.gz"
  sha256 "b01fba2c83149b4c34fe4dd6f84a6e24a59bbb87393db67c87123f4206e2e72c"
  license "MIT"

  depends_on :macos

  def install
    # The release archive ships the universal binary as "muse" (GoReleaser
    # names the universal binary after the build id). Install it as "mu",
    # which is the command name the README documents.
    bin.install "muse" => "mu"
  end

  test do
    assert_match "Muse", shell_output("#{bin}/mu version")
  end
end
