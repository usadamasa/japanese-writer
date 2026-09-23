<!-- writing-gate-disable process-leak pink-elephant -->

<!-- 検出対象の漏出パターンを例示として本文に持つため、該当ルールだけ無効化する｡ -->

# Tier 分類

proofreader subagent は検出した修正候補を必ず以下の 3 Tier のいずれかに分類する｡

proofreader は [[japanese-tech-writing]] の Phase 3 self-check の **subagent 側 (網羅視点)** に対応する｡
書き手の self-check (主観・気付き) が見落とすものを subagent が網羅的にスキャンして拾う｡両者は重複が健全｡

## Tier 1: 直接 Edit (黙って修正)

**判定基準**: 機械的に正解が一意で､ 文意への影響がない or 軽微｡ 事実関係への影響もない｡

| 修正内容 | 例 |
|---------|-----|
| textlint auto-fix 可能 rule | 二重否定の除去､ 冗長表現 (lint 結果の `applied_fixes` に入っているもの)､ 長音記号統一 |
| タイポ・表記揺れ | 全角英数 → 半角､ 句読点の半角化 |
| **英文略号の和訳** (japanese-tech-writing Phase 2.9) | `e.g.` → 「例えば」 / `i.e.` → 「すなわち」 / `etc.` → 「など」 / `cf.` → 「参照: 」 |
| **数字・日付表記の半角化** (japanese-tech-writing Phase 2.7) | 全角アラビア → 半角 / 「午前10時」 → 「10:00」 / 全角チルダ `～` → 波ダッシュ `〜` |
| **リンクテキスト「こちら」「click here」検出** (japanese-tech-writing Phase 2.10) | grep ベース検出 → 提案差し替え (1 候補に絞れるなら Tier 1、複数候補なら Tier 2) |

**動作**: textlint 由来のものは親が Step 0 で適用済み｡ それ以外 (domain_check 由来の確定置換など) を
subagent が `Edit` ツールで適用する｡ 親には件数と適用箇所の行番号のみ報告｡

## Tier 2: 提案 (親で executing-plans 風レビュー)

**判定基準**: 候補は出せるが､ 選択に文脈判断が要る｡ 構造変更を含む｡

| 修正内容 | 例 |
|---------|-----|
| textlint 検出のみ rule (auto-fix なし) | 「することができる」 (文脈次第で「できる」化)､ 一文長すぎ (Phase 2.5: 60字目安/100字上限)､ 漢字過剰 |
| japanese-tech-writing 規範違反 (Phase 2) | パラグラフ分割 (Phase 2.2)､ 段落の論証順序 (Phase 2.2)､ 前方参照の位置 (Phase 2.2)､ 譲歩の根拠付け (Phase 2.2)､ 受動態多用 (Phase 2.4: 能動態を既定とする)､ 箇条書きの lead-in コロン違反 (Phase 2.8) |
| **japanese-tech-writing Phase 1 違反** | 重点先行できてない (節が背景から始まる、Phase 1.4) → 構造変更提案 / scope 節欠落 (ADR/Design Doc/Stock で必須、Phase 1.3) → 追加提案 |
| **japanese-tech-writing Phase 2 新規節違反** | 一文 100 字超 (Phase 2.5)・読点 3 個以上 / 平易語の選択 (Phase 2.6) / 「あなた」使用 (Phase 2.4 で却下されている) |
| **制作過程の漏出** ([[sanitize-artifacts]]) | 依頼への応答 (「ご要望に従い」) / 修正の履歴 (「前回の版と異なり」) / 制約の宣言 (「〜は使わない」) / 方針転換の経緯注記 (「以前は X だったが現在は不要」) → 削除案、 または制約を満たした記述への置換案 |

**動作**: subagent が diff/箇条書きで親に return｡ 親は Stage 2 で apply 対象 ID を受領後 `Edit` 適用｡

## Tier 3: 警告のみ (修正しない)

**判定基準**: 機械的修正不可能｡ 論証構造や事実確認が必要｡

| 修正内容 | 例 |
|---------|-----|
| 論証の厳密さ違反 (japanese-tech-writing Phase 2.3) | 因果の機構未説明､ 譲歩の宙吊り､ 「同じ」 で異種をくくる､ 前方参照が回収されていない |
| **Phase 3 self-check 項目に該当する論理構造違反** | 重点先行ができてない (Phase 3.1 項目1) / scope 節欠落 (Phase 3.1 項目2、subagent が必須の文書で判定) / 段落先頭文で論証が追えない (Phase 3.1 項目4) / 論証の一方向性違反 (Phase 3.1 項目5) / 例が主張の範囲を支えてない (Phase 3.1 項目6) |
| 規範違反だが修正方向が複数あって絞れない | 「視点と語り」 違反 (受動態が連続) など |
| **制作過程の漏出のうち読者の要否が割れるもの** ([[sanitize-artifacts]]) | 回避した手段への言及が ADR の `Alternatives considered` に相当しうる / 断り書きが読者に必要か判断できない |

**動作**: subagent が警告のみ return｡ 親は Stage 1 で警告表示｡ ユーザーが Stage 1.5 (deep-dive) で詳細を要求できる｡ Tier 3 自体は apply 対象外｡

**Phase 3 self-check との重複**: 上記「Phase 3 self-check 項目に該当する論理構造違反」は、 書き手が Phase 3 で頭で点検済みであっても subagent が再度スキャンする｡書き手の主観が見落としたものを拾うのが目的｡警告だけ出して、 ユーザーが必要なら deep-dive で詳細確認できる｡

## Tier 判定の優先順位

1. 修正内容が **domain_check 由来** (ドメイン固有の用語表・事実情報) なら:
   - `domain_check.tiering_ref` の分類に従う｡本ファイルの基準では判定しない｡
   - `domain_check` 未指定なら、このドメインチェック自体を行わない (`skipped` に記録)｡

2. 修正内容が **textlint** 検出なら:
   - lint 結果の `applied_fixes` に入っている (autofix 済み) → **Tier 1**
   - `remaining_issues` に残っている → **Tier 2** (修正方向が複数あって絞れないなら Tier 3)

3. 修正内容が **japanese-tech-writing** 規範違反なら:
   - **Phase 1 違反** (文書の種類の判定不整合・scope 節欠落・重点先行できてない) → 構造変更を伴うので **Tier 2** (修正方向が複数の場合は Tier 3)
   - **Phase 2 形式的** (整形ルール: ダッシュ等、英文略号の和訳、数字日付の半角化) → 一意に修正できるなら **Tier 1**、文脈判断が要るなら **Tier 2**
   - **Phase 2 構造的** (段落分割､ 論証順序、能動態化、リスト形式変換) → **Tier 2**
   - **Phase 2.3 論証の厳密さ違反** → **Tier 3**
   - **Phase 2.4 視点と語り の違反** (「あなた」使用は Phase 2.4 で却下) → **Tier 2** (置換候補が一意なので)、その他は **Tier 3**
   - **Phase 3 self-check 項目** (重点先行・段落先頭文・論証一方向・例支持) → **Tier 3** (網羅視点で警告のみ)

4. 修正内容が **sanitize-artifacts** 違反 (制作過程の漏出) なら:
   - 削除または置換の案が一意に書ける → **Tier 2**
   - 読者がその情報を必要とするか判断が割れる → **Tier 3**
   - Tier 1 には落とさない｡ 漏出の判定は読者の要否に依存し、 機械的な一意解が無いため

5. 修正内容が **style_check 由来** (差し込まれた文体 skill の違反) なら:
   - 文体の指紋 (語尾､ 接続詞) のずれ → **Tier 2**
   (Tier 1/3 該当なし — 個人文体は機械修正の対象外であり､ 警告のみで apply 候補にならないケースは Tier 2 で `suggestion` 不完全として扱う)
   - `style_check` 未指定なら、このチェック自体を行わない (`skipped` に記録)｡

## 文書の種類別の優先度調整 (japanese-tech-writing Phase 1.2 マトリクスに従う)

medium と「衝突時の優先」によって、 同じ違反でも Tier が変わる:

- **fofr 優先の文書** (ADR / Design Doc / Stock 文書 / 書籍の章): Phase 1 違反 (重点先行・scope 節) と Phase 2 規範違反を **厳しく** 適用｡ 個人文体由来の「推量・自虐」検出は Tier 2/3 にしない (本スキルがそもそも Phase 1.2 で抑制すべきと判定済み)｡
- **文体優先の文書** (Slack / Flow 文書 / 議事録): Phase 2 規範違反 (一文長さ・能動態徹底・60 字制限) は **ゆるめて** 適用｡ Tier 2 → Tier 3 への格下げ、 または skipped 扱いを検討｡`style_check` が無ければ、ゆるめずに fofr 優先の文書と同じ強さで見る｡
- **文脈で判断する文書** (blog / 長文記事): Phase 1.2 で決まった優先度に従う｡ Phase 1 の判定が不明なら wiki 系の中庸扱い｡

medium から文書の種類への対応:

| medium | 文書の種類の判定 |
|--------|---------|
| `wiki` / `docs` | 本文の型で決める｡ADR / Design Doc / 運用手順なら fofr 優先の文書、議論メモ・週次なら文体優先の文書 |
| `note` | 文体優先の文書 (思考のスケッチ段階) |
| `short-draft` | 短文｡Phase 2 の段落・論証規範は適用しない |
