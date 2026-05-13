# typed: false
# frozen_string_literal: true

class Gobird < Formula
  desc "Twitter/X CLI tool and Go client library"
  homepage "https://github.com/mudrii/gobird"
  version "26.05.13"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/mudrii/gobird/releases/download/26.05.13/gobird_26.05.13_darwin_amd64.tar.gz"
      sha256 "87368223acd945918ee0939e4318c9814bf0192772bd04e8dd9a49eb820c57a4"
    end
    if Hardware::CPU.arm?
      url "https://github.com/mudrii/gobird/releases/download/26.05.13/gobird_26.05.13_darwin_arm64.tar.gz"
      sha256 "09358f23254b30d6a07e778d6f6349f139e7dfc1601c7a718f6ccd1c79849a41"
    end
  end

  on_linux do
    if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.05.13/gobird_26.05.13_linux_amd64.tar.gz"
      sha256 "bdf648f1032c69efa5789bfd11c430144ca42dbfc5e93e1374d835b2ea147053"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.05.13/gobird_26.05.13_linux_arm64.tar.gz"
      sha256 "49fc35d7d26f43130f6c1b06a302449e71926d008a6fcbb43121ced41a15991c"
    end
  end

  def install
    bin.install "gobird"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gobird --version")
  end
end
