---
name: crediting-sources
description: >-
  Use when 他の作者の skill・gist・ブログ記事・X の投稿・書籍・リポジトリを参考にして、
  このリポジトリの skill・agent・hook・writing-gate のルール・prh のルールを足す・直すとき｡
  「これを参考に」「この gist を取り込んで」「この投稿の考え方を入れて」と URL や書名を渡されたとき、
  原典を翻案した・移植した・一部を採り入れたときに使う｡
---

# crediting-sources

外部の資料から取り込んだものは､2 箇所へ書く｡
取り込んだ先の `## 出典` 節と､リポジトリ直下の `ACKNOWLEDGMENTS.md` になる｡
`ACKNOWLEDGMENTS.md` は README から張っている一覧で､詳細は各 skill の `## 出典` 節が持つ｡

## 対象

- 他の作者の skill・agent・gist・記事・X の投稿・書籍・公的資料の規範や手順を､翻案・移植・一部採用したとき
- 原典の考え方を出発点に､writing-gate のルールや hook を設計したとき (本文の転記が無くても対象)

対象外:

- 実行時に依存するだけのツールやライブラリ (textlint の preset､Go のモジュール)
- 自分のリポジトリ (`usadamasa/*`) から移したもの

## 書く場所

| 取り込んだ先 | `## 出典` 節を書く場所 |
| ---- | ---- |
| skill | `skills/<name>/SKILL.md` の末尾｡`references/` の 1 ファイルだけに効くなら､そのファイルの末尾 |
| writing-gate のルール・hook | `writing-gate/README.md` の末尾 |
| prh のルール | 呼び出し元の skill (`skills/textlint-check/SKILL.md`) の末尾 |

`## 出典` 節が無ければ新しく作る｡既にあるなら項目を足す｡

writing-feedback の Step 2 は､ルールのコメントに指摘の出所を書かせない｡これは手元の文書やレビューの情報を
ルールセットに残さないための決まりで､公開された原典の出典とは別の話になる｡`rules.json` や prh の
コメントには出所を書かず､上の表の `## 出典` 節と `ACKNOWLEDGMENTS.md` に書く｡

## ACKNOWLEDGMENTS.md の 1 行

外部の作者のものは「他の作者の skill・gist・投稿」表へ､書籍と公的資料は「書籍・公的資料」表へ足す｡

```markdown
| <作者> の <種類> (<原典での呼び名>) <URL> | <gist / skill / X の投稿 / 記事> | [<取り込んだ先>](<パス>) の <節> | <翻案 / 移植 / 一部採用> | <ライセンス> |
```

- 原典での呼び名は原典の表記を使う｡題名を自分で付けない
- ライセンスは原典のリポジトリの LICENSE を見る (`gh api repos/<owner>/<repo>/license --jq .license.spdx_id`)｡
  404 や gist なら「表記なし」､投稿は「-」
- MIT・Apache-2.0 などで本文を転記したときは､原典の著作権表示とライセンス文を取り込んだ先へ残す
- 同じ原典を別の場所へも取り込んだら､新しい行は作らず既存の行の「取り込んだ先」へ足す

## 確認

```bash
rg -n '<原典の URL>' ACKNOWLEDGMENTS.md skills agents writing-gate
```

`ACKNOWLEDGMENTS.md` と取り込んだ先の `## 出典` 節の 2 箇所に出れば済んでいる｡
