# pink-elephant

却下・削除した案の痕跡を検出する｡既定の severity は error｡

読者は捨てた案を知らないため､削除跡だけが目に入る｡

## 判定の範囲

- パターンの正本は `sanitize-artifacts` skill の「検出対象」表｡ここではそのうち逐語で拾えるものを機械化した｡
  意味・注意配分・視覚の層は機械では拾えないため､同 skill の点検が別に要る｡
- 引用行・コードブロック・インラインコード・フロントマター・HTML コメントは見ない (README の「漏出フレーズ」節)｡

## 設定

`rules.json` の `phrase_rules` にある `id: pink-elephant` の `patterns` (部分一致) と `regexps` (Go の正規表現) で決まる｡
