# japanese-writer

日本語の技術文書を書く・校正するための Claude Code plugin。
書くときは構成の規範 (japanese-tech-writing) が導き、書き上げた文書は textlint と subagent で添削する (proofread)。
受けた指摘はルールへ移し (writing-feedback)、セッションを閉じるときに Stop hook の writing-gate が点検する。
Product Brief (解決策を決める前に、なぜその問題に取り組むかを書く文書) を書くときは、
構成の規範に Brief の項目の型と点検の観点を足す (writing-product-briefs)。

## インストール

前提は Node.js (`npx`)、`jq`、Go。install の後に手で実行するコマンドは無い。

構成と開発の手順は [.claude/CLAUDE.md](.claude/CLAUDE.md) にある。

## ライセンス

MIT
