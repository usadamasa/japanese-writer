# japanese-writer

日本語の技術文書を書く・校正する Claude Code の plugin。

この plugin の利用者は usadamasa 本人だけ。
不特定多数の利用者の手元環境 (Go のバージョン、`GOTOOLCHAIN` の設定など) との互換は考慮しない。

文章の指摘 (crit のコメント、言い回しの修正) の扱いで「inbox」と言われたら、
writing-feedback skill が書き込む retrospective の inbox を指す。

依頼文やメモが skill を名指ししていたら、その skill を読んでから依頼の語を解釈する。

## 構成

入口は 2 つ。書くときは `japanese-tech-writing` が構成から推敲まで導き、書き上がったら
`/japanese-writer:proofread` で添削する。各部品の役割は、それぞれの SKILL.md と agent の定義にある。

```text
ユーザー
├─ /japanese-writer:setup ── 自動の準備が失敗したとき、crit をリンクするとき
├─ 書く ── japanese-tech-writing (Phase 1 構成 → 2 執筆 → 3 推敲)
│             └─ Phase 3 の最後に proofread を呼ぶ
├─ Product Brief を書く・点検する ── writing-product-briefs (項目の型と点検の観点)
│    └─ 文章は japanese-tech-writing に従い、共有する前に proofread を呼ぶ
├─ /japanese-writer:proofread ── 添削の入口
│    ├─ textlint-check (機械点検と autofix)
│    └─ proofreader (subagent)
│         ├─ japanese-tech-writing (規範として読む)
│         ├─ sanitize-artifacts (漏出の観点として読む)
│         ├─ style_check で差し込んだ文体 skill (任意)
│         └─ domain_check で差し込んだ用語 skill (任意)
├─ /japanese-writer:writing-feedback ── 言い回しを直されたとき
│    ├─ crit の approve 後の prompt (crit/on_finish_approved.md) からも呼ばれる
│    └─ 構成・文脈の指摘は inbox へ書くだけにする
├─ /japanese-writer:writing-retrospective ── 思い出したとき、dream を回すとき
│    └─ inbox・crit の記録・セッションログを棚卸しして japanese-tech-writing の規範へ入れる
├─ /japanese-writer:gate-off, gate-on ── このセッションの間だけ Stop hook の点検を止める・戻す
├─ (自動) SessionStart hook: writing-gate を data ディレクトリへビルド ── セッションを開くとき
└─ (自動) Stop hook: writing-gate ── セッションを閉じるとき
```

writing-gate の設計とルールの置き場は [writing-gate/README.md](../writing-gate/README.md) にある。
準備が失敗したときの直し方は setup skill にある。

## 開発

plugin を使う前提 ([README.md](../README.md) のインストール節) に加えて、次を用意する。

- flakes を有効にした [Nix](https://nixos.org/)。Go・task・golangci-lint・shellcheck・bats・jq・
  [jv](https://github.com/santhosh-tekuri/jsonschema) は `flake.nix` の devShell で揃え、バージョンは `flake.lock` が固定する
  nixpkgs で決まる。nixpkgs に無い jv は `flake.nix` の中でソースからビルドする。CI も同じ flake を使う
- [direnv](https://direnv.net/) と [nix-direnv](https://github.com/nix-community/nix-direnv)。`.envrc` が
  `use flake` で devShell を読み込むので `direnv allow` する
- Claude Code CLI (`task validate` が `claude plugin validate` を呼ぶ)

`go.mod` の `go` ディレクティブは devShell の `go_1_27` と系列を揃え、`GOTOOLCHAIN=local` でずれたら失敗させる。
`go` 行は利用者の手元で SessionStart hook がビルドするときの下限も兼ねるので、セキュリティ修正のバージョンに合わせる。
ツールのバージョンを上げるときは `nix flake update` で `flake.lock` を更新する。
Go は nixpkgs のバージョンが修正バージョンに届かない間、`flake.nix` の `devGo` が go.dev の配布バイナリへ差し替え、
jv も同じ Go でビルドする。

CI と同じ確認は task で手元でも通せる。

| コマンド | 内容 |
| ---- | ---- |
| `task build` | `bin/writing-gate` をビルドする |
| `task test` | ビルドしてから Go のテストと bats を実行する |
| `task lint` | golangci-lint と shellcheck を実行する |
| `task validate` | `claude plugin validate --strict` と、JSON / YAML が宣言した schema での検証を実行する |

参考にした・翻案した skill・gist・書籍は [ACKNOWLEDGMENTS.md](../ACKNOWLEDGMENTS.md) に載せる。

## issue と PR のラベル

種類・優先度・epic の付け方は `github-ops:labeling-github-issues` skill に従う。機能の area はこの表から選ぶ。

| area | 含むもの |
| ---- | ---- |
| `area: writing-gate` | Stop hook の writing-gate (Go の CLI、phrase rule、gate-off / gate-on) |
| `area: textlint` | textlint-check と同梱の textlint config、prh のルール |
| `area: proofread` | proofread skill、proofreader subagent、sanitize-artifacts |
| `area: tech-writing` | japanese-tech-writing の文章規範、writing-product-briefs |
| `area: feedback` | writing-feedback、writing-retrospective、crit の approve 後の prompt |
| `area: setup` | setup skill、SessionStart hook による writing-gate のビルドと data ディレクトリの準備 |
| `area: dev` | CI、lint、テスト、Taskfile、flake、保守用の skill と docs |

移設を取りやめた agy-delegation (#19) には area を付けない。

- PR の本文に、閉じる issue を `Closes #N` で 1 行ずつ書く。pr-labels workflow (gh-manage の reusable workflow の呼び出し) が issue の種類と area のラベルを PR へ写す。
- issue を閉じない PR は、`gh pr create --label` で種類と area のラベルを付けて作る。選び方は `github-ops:labeling-github-issues` skill。
