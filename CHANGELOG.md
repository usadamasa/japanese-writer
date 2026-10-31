# Changelog

## [2026.1031.0](https://github.com/usadamasa/japanese-writer/compare/2026.1030.4...2026.1031.0) - 2026-10-31

- member-name-list で「と」でつないだ識別子の列挙を拾う by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/100

## [2026.1030.4](https://github.com/usadamasa/japanese-writer/compare/2026.1030.3...2026.1030.4) - 2026-10-30

- 内部用語と主語の無い動詞を拾うルールを足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/98

## [2026.1030.3](https://github.com/usadamasa/japanese-writer/compare/2026.1030.2...2026.1030.3) - 2026-10-30

- 名前の列挙と本数を拾うルールを足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/96

## [2026.1030.2](https://github.com/usadamasa/japanese-writer/compare/2026.1030.1...2026.1030.2) - 2026-10-30

- ai-words-ja の辞書から選んだ語彙を prh-prose へ取り込む by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/94

## [2026.1030.1](https://github.com/usadamasa/japanese-writer/compare/2026.1030.0...2026.1030.1) - 2026-10-30

- prh-prose に検査を主語にした「効く」と「効き目」を登録する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/92

## [2026.1030.0](https://github.com/usadamasa/japanese-writer/compare/2026.1026.0...2026.1030.0) - 2026-10-30

- chore(deps): Bump Songmu/tagpr from 1.21.3 to 1.21.4 by @dependabot[bot] in https://github.com/usadamasa/japanese-writer/pull/88
- chore(deps): Bump nixpkgs from `02c087f` to `3ec6363` by @dependabot[bot] in https://github.com/usadamasa/japanese-writer/pull/89

## [2026.1026.0](https://github.com/usadamasa/japanese-writer/compare/2026.1021.0...2026.1026.0) - 2026-10-26

- chore(deps): Bump actions/create-github-app-token from 3.2.0 to 3.3.0 by @dependabot[bot] in https://github.com/usadamasa/japanese-writer/pull/79
- chore(deps): Bump actions/checkout from 6.0.3 to 7.0.1 by @dependabot[bot] in https://github.com/usadamasa/japanese-writer/pull/76
- chore(deps): Bump actions/upload-artifact from 7.0.1 to 7.0.2 by @dependabot[bot] in https://github.com/usadamasa/japanese-writer/pull/77
- chore(deps): Bump Songmu/tagpr from 1.20.3 to 1.21.3 by @dependabot[bot] in https://github.com/usadamasa/japanese-writer/pull/78
- go.mod に toolchain go1.26.10 を足し、hook のビルドを修正版の Go にする (GO-2026-6809) by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/81
- writing-gate: 作業の進行状況の注記を拾う in-progress-note を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/82
- setup-nix を gh-manage の共有 action へ移行する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/84
- ci: pull_request の types に edited を足し、base の付け替えで走り直すようにする by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/85
- crit レビューの言い回し指摘を prh / writing-gate のルールへ登録 by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/86

## [2026.1021.0](https://github.com/usadamasa/japanese-writer/compare/2026.1018.1...2026.1021.0) - 2026-10-21

- devShell の Go を 1.26.10 に上げる (GO-2026-6809) by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/73
- Dependabot の設定を足す (gomod・github-actions・nix) by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/74

## [2026.1018.1](https://github.com/usadamasa/japanese-writer/compare/2026.1018.0...2026.1018.1) - 2026-10-18

- fix(setup): 今の版と同じ内容を指す crit のリンクをリンク済みとする by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/71

## [2026.1018.0](https://github.com/usadamasa/japanese-writer/compare/2026.1017.1...2026.1018.0) - 2026-10-18

- setup.sh に書き込みを伴わない点検モード --check を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/69

## [2026.1017.1](https://github.com/usadamasa/japanese-writer/compare/2026.1017.0...2026.1017.1) - 2026-10-17

- feat(textlint): 他 skill の節名を指す参照を prh-prose で拾う by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/62
- fix(writing-gate): gate の指摘をセッション中に書いた行へ絞る by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/64
- fix(writing-gate): 同じ指摘を 1 回だけ出し、gate-off でセッション単位に止める by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/66

## [2026.1017.0](https://github.com/usadamasa/japanese-writer/compare/2026.1011.0...2026.1017.0) - 2026-10-17

- writing-gate: 本文に書いた実測日の注記を検出する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/51
- ci: GitHub Actions の固定を pinact から gh-actions-lock へ移行する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/56
- ci: gh-actions-lock を v0.2.0 へ上げ、ci-status-check の形を揃える by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/57
- Crit レビューの言い回し指摘を prh / writing-gate のルールへ登録する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/58
- 開発ツールの管理を aqua から nix の devShell へ移す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/60
- feat(textlint): foreground の直訳「前景」を prh-prose で拾う by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/61

## [2026.1011.0](https://github.com/usadamasa/japanese-writer/compare/2026.1010.1...2026.1011.0) - 2026-10-11

- writing-gate: inline-enumeration で括弧内の読点区切りの列挙を拾う by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/49

## [2026.1010.1](https://github.com/usadamasa/japanese-writer/compare/2026.1010.0...2026.1010.1) - 2026-10-10

- chore(aqua): golangci-lint を v2.14.0 に上げる by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/46
- proofread: Tier 2 の適用条件に意味の保持と情報の不増補を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/45
- docs: md の句読点を全角に揃える by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/48

## [2026.1010.0](https://github.com/usadamasa/japanese-writer/compare/2026.1004.0...2026.1010.0) - 2026-10-10

- prh: 「正本」禁止ルールの助詞カバレッジを広げる by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/42
- 文書の立場 (勧め・決まり・説明) で文末を揃える規範を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/44

## [2026.1004.0](https://github.com/usadamasa/japanese-writer/compare/2026.0930.0...2026.1004.0) - 2026-10-04

- proofread の textlint を毎回実行し、Tier 1 を報告から外す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/27
- docs: 参考にした外部の資料を ACKNOWLEDGMENTS.md にまとめる by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/29
- prh-prose に比喩動詞と前置きのフィラーの検出を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/36
- writing-gate: 太字として表示されない ** を bold-not-rendered で検出する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/38
- fix(textlint-check)!: 句読点を全角へ揃え、文長の誤検出を直す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/41

## [2026.0930.0](https://github.com/usadamasa/japanese-writer/compare/2026.0929.0...2026.0930.0) - 2026-09-30

- prh-prose.yml に「射程」の検出ルールを追加 by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/25

## [2026.0929.0](https://github.com/usadamasa/japanese-writer/compare/2026.0928.0...2026.0929.0) - 2026-09-29

- prh: crit レビューで直された言い回しをルール化する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/23

## [2026.0928.0](https://github.com/usadamasa/japanese-writer/compare/2026.0927.0...2026.0928.0) - 2026-09-28

- 見出しの動詞終止形を検出する writing-gate ルールを追加する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/20
- docs(japanese-tech-writing): 技術的識別子は原語のまま書く規範を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/22

## [2026.0927.0](https://github.com/usadamasa/japanese-writer/compare/2026.0926.07...2026.0927.0) - 2026-09-27

- feat(release): 版の更新を tagpr に任せる by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/17

## [2026.0926.07](https://github.com/usadamasa/japanese-writer/commits/2026.0926.07) - 2026-09-26

- 名前の解決を「届く」で書く文を検出する prh ルールを足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/2
- 日本語の執筆・校正を支援する Claude Code plugin を収録する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/1
- ci: GitHub Actions で Go・plugin・schema・シェルの検証を回す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/3
- prh: 動く/動かないの曖昧さを検出し、別セッション引き継ぎを案内する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/4
- crit approve 後 prompt を writing-feedback skill の規範に合わせる by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/5
- fix(proofread): Tier 2 を問い合わせずに適用して報告する流れにする by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/7
- feat(setup): 利用者向けの setup skill を追加する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/8
- marketplace の定義を agents-marketplace へ移す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/9
- prh: 「焼く」をイメージ・バイナリへの格納の比喩に使わないルールを追加する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/6
- 版を calver にし、writing-gate を data ディレクトリへ自動ビルドする by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/11
- 曖昧な動詞のprhルールに「引かれる」「起きているか」を足す by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/13
- claude-config crit の言い回し指摘を writing-feedback ルールへ登録する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/14
- feat(prepare-data): textlint-run.sh を CLAUDE_PLUGIN_DATA へ複製する by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/15
- fix(textlint-run): fix パスの空 stdout を JSON 妥当と誤判定しないようにする by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/12
- feat(textlint-run): plugin の bin/ へ移して bare name で呼ぶ by @usadamasa in https://github.com/usadamasa/japanese-writer/pull/16
