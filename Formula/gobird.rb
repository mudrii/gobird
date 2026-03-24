# typed: false
# frozen_string_literal: true

class Gobird < Formula
  desc "Twitter/X CLI tool and Go client library"
  homepage "https://github.com/mudrii/gobird"
  version "26.03.24"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_darwin_amd64.tar.gz"
      sha256 "a5b95d41207a6f5d872eca36c6e68f16913d43006ac989d46ff4f7320e19d88f"
    end
    if Hardware::CPU.arm?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_darwin_arm64.tar.gz"
      sha256 "365367f3174b8534156747618710706e29d2341e6352469b5470068565356852"
    end
  end

  on_linux do
    if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_linux_amd64.tar.gz"
      sha256 "e758591ccda0b0524926c2416cbaf77591ee334a5ecf3cfb042dd84605e7aaeb"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_linux_arm64.tar.gz"
      sha256 "f4c387d2c455e59afc7ec704ac76ccdd2b80e34c4dafd1b31191d742f035ceb4"
    end
  end

  def install
    bin.install "gobird"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gobird --version")
  end
end
