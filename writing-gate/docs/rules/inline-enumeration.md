<!-- writing-gate-disable inline-enumeration -->

# inline-enumeration

説明文の中で､括弧内の「A / B / C」や「A + B + C」で構成要素を並べている書き方を検出する｡既定の severity は warn｡

読者は何をするかを知りたいのであって､部品の一覧は要らない｡部品の一覧が要るなら表か箇条書きへ出す｡

## 判定の範囲

- 引用行・コードブロック・インラインコード・フロントマター・HTML コメントは見ない (README の「漏出フレーズ」節)｡

## 設定

`rules.json` の `phrase_rules` にある `id: inline-enumeration` の `regexps` (Go の正規表現) で決まる｡
