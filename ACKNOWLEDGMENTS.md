# 謝辞

この plugin の skill・hook・ルールは､次の資料を参考にし､または翻案して作った｡
各 skill の `## 出典` 節には､原典のどの部分をどこへ取り込んだかの詳細がある｡

## 他の作者の skill・gist・投稿

| 出典 | 種類 | 取り込んだ先 | 取り込み方 | ライセンス |
| ---- | ---- | ---- | ---- | ---- |
| k16shikano の gist (論証) <https://gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d> | gist | [japanese-tech-writing](skills/japanese-tech-writing/SKILL.md) の Phase 2/3 | 翻案 (骨格) | 表記なし |
| k16shikano の gist (Cognitive Rhythm) <https://gist.github.com/k16shikano/eb2929f13ed19c97188393d297be8432> | gist | [japanese-tech-writing](skills/japanese-tech-writing/SKILL.md) の Phase 2.17・3.1､[cognitive-rhythm.md](skills/japanese-tech-writing/references/cognitive-rhythm.md) | 翻案 (骨格) | 表記なし |
| fofr の gist (GOV.UK / GDS の翻案) <https://gist.github.com/fofr/505e225f9bf5e839d30c12ba6bfa0be2> | gist | [japanese-tech-writing](skills/japanese-tech-writing/SKILL.md) の Phase 1､Phase 2.5〜2.10 | 翻案 (一部ルールは不採用) | 表記なし |
| kotek-7/dotfiles の sanitize-artifacts <https://github.com/kotek-7/dotfiles/blob/main/dot_agents/skills/sanitize-artifacts/SKILL.md> | skill | [sanitize-artifacts](skills/sanitize-artifacts/SKILL.md) | 日本語文書向けに翻案 | 表記なし |
| @Kashiko_AIart の投稿 <https://x.com/Kashiko_AIart/status/2091137586991645101> | X の投稿 | [sanitize-artifacts](skills/sanitize-artifacts/SKILL.md) の「却下した案の扱い (ピンクの象)」節､[writing-gate](writing-gate/README.md) の却下案の痕跡の検出 | 翻案 | - |
| @yugen_matuni の投稿 <https://x.com/yugen_matuni/status/2088251220452679951> | X の投稿 | [writing-feedback](skills/writing-feedback/SKILL.md)､[writing-gate](writing-gate/README.md) と Stop hook (執筆後の機械点検の層) | 翻案 | - |
| nanaism の skill (yomiyasu) <https://github.com/nanaism/yomiyasu> | skill | [textlint-check](skills/textlint-check/SKILL.md) の prh ルール (比喩動詞・前置きのフィラー・定型の結びの検出)､writing-gate の [`bold-not-rendered`](writing-gate/docs/rules/bold-not-rendered.md) (太字として表示されない `**` の検出) | 一部採用､writing-gate は移植 | MIT |

## 書籍・公的資料

| 出典 | 取り込んだ先 |
| ---- | ---- |
| 文化庁「公用文作成の考え方」(令和4年改定) | [japanese-tech-writing](skills/japanese-tech-writing/SKILL.md) の Phase 1 の読み手意識､Phase 2 の平易さと数字・日付の書き方 |
| 木下是雄『理科系の作文技術』(中公新書, 1981) | [japanese-tech-writing](skills/japanese-tech-writing/SKILL.md) の Phase 1 の重点先行・パラグラフライティング・事実と意見の区別 |
| 結城浩『数学文章作法 推敲編』 | [japanese-tech-writing](skills/japanese-tech-writing/SKILL.md) の Phase 3 の self-check の形式､Phase 2.5 の一文の長さの閾値 |

## 追記の手順

新しい出典を取り込んだら､取り込んだ先の `## 出典` 節とこのファイルの両方へ書く｡
手順は `.claude/skills/crediting-sources/SKILL.md` にある｡
