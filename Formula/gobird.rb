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
      sha256 "7dd8574bf46922e6a65139951bdd097c56e6767f76590ab0a0bfd8db2c5b6008"
    end
    if Hardware::CPU.arm?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_darwin_arm64.tar.gz"
      sha256 "fb954f22bce7f13d95f16fc8a4583164253f09dcb27600e05db38ad797a82154"
    end
  end

  on_linux do
    if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_linux_amd64.tar.gz"
      sha256 "2be269823f53d047cba67b5e4a285284d8d42bfedadbb5bdf3e813a0e9f86ffe"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.03.24/gobird_26.03.24_linux_arm64.tar.gz"
      sha256 "901215e2643975ee1735e07842ba2f78463b402a01a4176255de7125b542bb65"
    end
  end

  def install
    bin.install "gobird"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gobird --version")
  end
end
