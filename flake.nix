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
      # nixos-unstable の go_1_27 は 1.27.3 で、net/http の MaxBytesReader のデータ競合 (CVE-2026-97054) と
      # cgo 経由でビルド時に任意コードが実行される cmd/go などの脆弱性 (CVE-2026-97033〜97043) が残る。
      # channel の go_1_27 が 1.27.4 より古いときだけソースを差し替える。Dependabot の nix PR が 1.27.4 以上の
      # nixpkgs を取り込んだ時点で channel の go_1_27 に戻り、以後のパッチ更新は flake.lock と一緒に上がる。
      # 戻った後の差し替えの分岐は使われないので、そのとき消す
      devGo =
        pkgs:
        if pkgs.lib.versionOlder pkgs.go_1_27.version "1.27.4" then
          pkgs.go_1_27.overrideAttrs (old: rec {
            version = "1.27.4";
            src = pkgs.fetchurl {
              url = "https://go.dev/dl/go${version}.src.tar.gz";
              hash = "sha256-c8+4VxrsDAEXdofbhxY9Rqog6XeWcncokaoXZU2aYJk=";
            };
          })
        else
          pkgs.go_1_27;
    in
    {
      devShells = forAllSystems (
        pkgs:
        let
          go = devGo pkgs;
          buildGoModule = pkgs.buildGoModule.override { inherit go; };

          # nixpkgs に無い santhosh-tekuri/jsonschema の CLI。cmd/jv は独立した module
          # で、リポジトリ直下の go.work は vendor できないので外す
          # (ライブラリは cmd/jv/go.mod の require の版になる)
          jv = buildGoModule rec {
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
        in
        {
          default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              # go.mod の go ディレクティブと同じ系列。ずれたら GOTOOLCHAIN=local で即座に失敗させる。
              # jv も同じ go でビルドする
              go
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
