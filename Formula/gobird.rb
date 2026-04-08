# typed: false
# frozen_string_literal: true

class Gobird < Formula
  desc "Twitter/X CLI tool and Go client library"
  homepage "https://github.com/mudrii/gobird"
  version "26.04.08"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/mudrii/gobird/releases/download/26.04.08/gobird_26.04.08_darwin_amd64.tar.gz"
      sha256 "253723374f2a2588dbb884cbb79677736cd7fd0db4f7c6c92a10fa5726b8fc90"
    end
    if Hardware::CPU.arm?
      url "https://github.com/mudrii/gobird/releases/download/26.04.08/gobird_26.04.08_darwin_arm64.tar.gz"
      sha256 "641f4ffabc39991b5c07615791dd8c04fa25efa9a14c6e44244ced12f076f37e"
    end
  end

  on_linux do
    if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.04.08/gobird_26.04.08_linux_amd64.tar.gz"
      sha256 "e612db68f3c0fe83fe70c87c0dc11ceb5876f2373897bed8e786057a558bd203"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/mudrii/gobird/releases/download/26.04.08/gobird_26.04.08_linux_arm64.tar.gz"
      sha256 "afcdcb19d67e47ca6b7a0995790d700578886a6243c6f2c7ce5f276c15ea8acf"
    end
  end

  def install
    bin.install "gobird"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/gobird --version")
  end
end
