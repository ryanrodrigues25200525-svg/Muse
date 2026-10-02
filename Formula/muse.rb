class Muse < Formula
  desc "CLI, TUI, and MCP controller for Apple Music on macOS"
  homepage "https://github.com/ryanrodrigues25200525-svg/Muse"
  url "https://github.com/ryanrodrigues25200525-svg/Muse/releases/download/v0.1.1/muse_0.1.1_darwin_universal.tar.gz"
  sha256 "4f7df7fd35259eb855b7bd21fe58434805aab56acc27f0c28bc8e4645a8b710d"
  license "MIT"

  depends_on :macos

  def install
    bin.install "mu"
  end

  test do
    assert_match "Muse", shell_output("#{bin}/mu version")
  end
end
