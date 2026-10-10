# sentence-ending-repeat

同じ語尾の文が段落内で 3 文以上続いていることを検出する。既定の severity は error。

## 判定の範囲

- 本文の段落だけを見る。箇条書きの語尾が揃うのは規範上むしろ推奨されるため (`japanese-tech-writing` Phase 2.8)。
- 段落 (空行と非本文行で区切る) をまたいだ連続は数えない。見出しを挟んだ「連続」は読み手に連続として届かない。
- 語尾がひらがなで終わらない行は数えない。語彙の列挙や記号の並びを拾ってしまうため。

## 設定

`rules.json` の `aggregates.sentence_ending_repeat` で変える。

| キー | 既定 | 意味 |
| ---- | ---- | ---- |
| `min_run` | 3 | 何文続いたら拾うか |
| `suffix_runes` | 3 | 語尾として比べる末尾の文字数 |
| `min_sentence_runes` | 6 | これより短い文は判定しない |
