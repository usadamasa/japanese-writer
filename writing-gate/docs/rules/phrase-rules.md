# phrase rules

`rules.json` の `phrase_rules` に並べた､語・句のパターンで拾うルール｡置換先を持たないため prh には載せられないものをここに置く｡

各ルールの何が問題か・どう直すかは､`rules.json` の `message` と `guidance` が正本になる｡
ルールは `writing-feedback` skill が足すので､この文書へルールごとの説明は書き写さない｡

## 判定の範囲

- 引用行 (`>`) は他人の文なので見ない
- コードブロック・インラインコード・フロントマター・HTML コメントは解析前に落とす
- `kinds` を書いたルールは､書いた種別の行だけに当たる｡省略すると引用以外の全行に当たる

## 設定

ルールごとに `rules.json` の次のキーで決まる｡

| キー | 意味 |
| ---- | ---- |
| `id` | ルール ID｡無効化の指定 (`writing-gate-disable <id>`) にも使う |
| `severity` | `error` なら Stop hook が完了を止める｡`warn` は報告だけ |
| `message` / `guidance` | 何が問題か / どう直すか |
| `patterns` | 部分一致で拾う文字列 |
| `regexps` | Go の正規表現 |
| `kinds` | 当てる行の種別｡`body`､`heading`､`list`､`table` から選ぶ |

## sanitize-artifacts との対応

`process-leak` (制作過程の痕跡) と `pink-elephant` (却下・削除した案の痕跡) のパターンの正本は､
`sanitize-artifacts` skill の「検出対象」表｡ここではそのうち逐語で拾えるものを機械化した｡
意味・注意配分・視覚の層は機械では拾えないため､同 skill の点検が別に要る｡
