# process-leak

成果物に残った制作過程の痕跡を検出する｡既定の severity は error｡

会話で受けた指示や依頼への言及は､会話を見ていない読者には意味のない一文として読まれる｡

## 判定の範囲

- パターンの正本は `sanitize-artifacts` skill の「検出対象」表｡ここではそのうち逐語で拾えるものを機械化した｡
  意味・注意配分・視覚の層は機械では拾えないため､同 skill の点検が別に要る｡
- 引用行・コードブロック・インラインコード・フロントマター・HTML コメントは見ない (README の「漏出フレーズ」節)｡

## 設定

`rules.json` の `phrase_rules` にある `id: process-leak` の `patterns` (部分一致) と `regexps` (Go の正規表現) で決まる｡
