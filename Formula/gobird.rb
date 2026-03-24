# typed: false
# frozen_string_literal: true

class Gobird < Formula
  desc "Twitter/X CLI tool and Go client library"
  homepage "https://github.com/mudrii/gobird"
  version "26.03.15"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/mudrii/gobird/releases/download/26.03.15/gobird_26.03.15_darwin_amd64.tar.gz"
      sha256 "7d58b71f7be6f2938d3bfc6147670c26fd7144b587551c618a912bbdcb1f1bcc"
    end
    if Hardware::CPU.arm?
      url "https://github.com/mudrii/gobird/releases/download/26.03.15/gobird_26.03.15_darwin_arm64.tar.gz"
      sha256 "3b7f2ebdce440739072be6f0e4a48215f4061a9dc4e1fe25db45ca502c514102"
    end
  end

  on_linux do
    if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.03.15/gobird_26.03.15_linux_amd64.tar.gz"
      sha256 "1931e6ac00980133a78d61c56532221c61ebf73cc9e6ed9dab4d6bf1fc25e02f"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.03.15/gobird_26.03.15_linux_arm64.tar.gz"
      sha256 "ead3296fb99705d8e23fc12e30bc22d7fe5bcbd7a97fe3aae140a1edd89d6344"
    end
  end

  def install
    bin.install "gobird"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gobird --version")
  end
end
