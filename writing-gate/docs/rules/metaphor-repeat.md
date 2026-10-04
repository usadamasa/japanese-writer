# metaphor-repeat

比喩の目印が文書内に 3 箇所以上あることを検出する｡同じ比喩を使い回していないかの確認を促す｡既定の severity は warn｡

## 判定の範囲

- 目印は `rules.json` の `metaphor_markers` に並べた語｡引用行は数えない｡
- 文書全体で 1 件にまとめ､最初に目印が出た行を指す｡

## 設定

`rules.json` の `aggregates.metaphor_repeat` で変える｡

| キー | 既定 | 意味 |
| ---- | ---- | ---- |
| `min_count` | 3 | 目印が何箇所あったら拾うか |
