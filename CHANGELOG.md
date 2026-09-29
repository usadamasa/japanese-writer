# Changelog

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
