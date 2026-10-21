{
  description = "japanese-writer の開発環境 (Go と lint・テスト・検証のツール)";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (
        pkgs:
        let
          # nixpkgs に無い santhosh-tekuri/jsonschema の CLI。cmd/jv は独立した module
          # で、リポジトリ直下の go.work は vendor できないので外す
          # (ライブラリは cmd/jv/go.mod の require の版になる)
          jv = pkgs.buildGoModule rec {
            pname = "jv";
            version = "6.0.3";
            src = pkgs.fetchFromGitHub {
              owner = "santhosh-tekuri";
              repo = "jsonschema";
              tag = "v${version}";
              hash = "sha256-KohVVD277qn+ZHJgJsn4EBNfxUZnsYX2AT8LA3vI5P4=";
            };
            modRoot = "cmd/jv";
            env.GOWORK = "off";
            vendorHash = "sha256-1vVdVqhUAcATwsPHXcwm4M6+jBuazDbxwYPDTHLOT1Y=";
          };

          # nixos-unstable の go_1_26 は 1.26.9 で、crypto/x509 の GO-2026-6809 (1.26.10 で修正) が残るので
          # go のソースだけを差し替える。go_1_26 が 1.26.10 以上になったらこの override を消して go_1_26 に戻す
          # (nix flake update だけでは消えない)
          devGo = pkgs.go_1_26.overrideAttrs (old: rec {
            version = "1.26.10";
            src = pkgs.fetchurl {
              url = "https://go.dev/dl/go${version}.src.tar.gz";
              hash = "sha256-/WPog7ovK/ypPBZGsx6+9j4rHk0rMXyXidl2po2B828=";
            };
          });
        in
        {
          default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              # go.mod の go ディレクティブと同じ版。ずれたら GOTOOLCHAIN=local で即座に失敗させる
              devGo
              go-task
              golangci-lint
              shellcheck
              bats
              jq
              jv
            ];
            env.GOTOOLCHAIN = "local";
          };
        }
      );
    };
}
