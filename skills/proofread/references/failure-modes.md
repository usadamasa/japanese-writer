# 失敗時挙動マトリクス

proofreader subagent や周辺ツールが失敗した時の挙動を定義する｡

## F1: textlint を実行できない

**検出**: 親が Step 0 で `textlint-check` を invoke したとき、 `textlint-run.sh` が非 0 で終了｡
npx / jq の不在、 config の不在、 config が参照する rule の解決失敗などが該当する｡

**親動作**: proofread skill 全体を error 返却で終了｡ subagent は dispatch しない｡ md ファイルは無変更｡

**表示**: `✗ proofread 動作不能 (textlint を実行できない)｡ md ファイルは無変更｡ 原因: {stderr の要約}`

**ユーザー指示**: `textlint-check` の 「失敗時」 節の表に従って前提条件を直してから再実行する｡

機械点検を欠いたまま添削を完了扱いにしない｡ 「textlint が動かないのに添削が通った」 状態は、
規範チェックだけ通った md を clean と誤認させる｡

**subagent 側**: 親が先に落ちるので通常は届かない｡ `lint_result_path` を渡されたのに `Read` が失敗した
場合だけ、 `failures` に `{"skill": "textlint-check", "reason": "lint_result_path を読めなかった"}` を
追加して続行する｡

## F2: Skill 読み込み失敗

**検出**: subagent が `Skill` ツールで `japanese-tech-writing` 等を読めなかった (symlink 切れ等)｡

**subagent 動作**: 該当 step を skip し `failures` に記録して続行｡

**親動作**: F1 と同様に Stage 1 の報告冒頭で警告｡

**ユーザー指示**: 該当 skill が読める状態に戻す (plugin なら再インストール、`style_check` / `domain_check` で
差し込んだ skill なら、その skill のデプロイ先を確かめる)｡

## F3: subagent dispatch 失敗

**検出**: 親が Agent ツールで proofreader dispatch 時に transient エラー、 quota 制限｡

**親動作**: 1 回までリトライ｡ 2 回連続失敗なら proofread skill 全体を error 返却で終了｡

**表示**: `✗ proofread 動作不能 (subagent dispatch 失敗 2 回連続)｡ md ファイルは無変更｡ 再実行するか skip するか指示してください`

**ユーザー指示**: 再実行 or skip｡

## F4: Edit 失敗 (Tier 1 apply 中)

**検出**: subagent が Step 6 で Edit 呼び出し時に失敗 (ファイル変更済み、 権限エラー等)｡

**subagent 動作**: 失敗位置以降の Tier 1 を停止｡ return JSON に以下追加:

```json
{
  "tier1_applied_count": N-1,
  "tier1_failed_at": {"id": "1-N", "reason": "..."},
  "tier1_pending": [
    {"id": "1-N", "rule": "...", "lines": [...]},
    {"id": "1-N+1", "rule": "...", "lines": [...]}
  ]
}
```

**親動作**: Stage 1 の報告に `⚠ Tier 1 の {N-1} 件まで apply 済 ｡ 以降は停止 (理由: ...)｡ 未適用分は Tier 2 として扱う` と表示し、 `tier1_pending` を Tier 2 リストに編入 (ID は `2-X` として振り直し) して親が適用する｡ 親でも Edit が失敗した ID は「Tier 2 未適用」に理由付きで載せる｡

## F5: subagent 判定異常 (大量変更)

**検出**: subagent が Step 6 完了後、 `tier1_change_ratio` (変更行数 / 全行数) が 0.3 を超えている｡

**subagent 動作**: revert は試みない (tools に Bash/git なし)｡ return JSON に以下追加:

```json
{
  "tier1_change_ratio": 0.45,
  "warning": "abnormal change ratio (>30%)｡ 親で git diff 確認・revert 判断を推奨"
}
```

**親動作**: Tier 2 の適用を止め、 報告冒頭に `⚠ Tier 1 が異常規模 (X% 変更) ｡ 親で git diff 確認、 revert 推奨` と表示｡ Tier 2 は全件「未適用」に載せ、 ユーザーに git status / git diff 確認と revert 判断を委ねる｡ 異常規模の上に Tier 2 を重ねると、 revert で戻す範囲が判別できなくなる｡

**ユーザー指示**: `git diff -- {file_path}` で確認後、 必要なら `git restore -- {file_path}` で revert｡
どちらも cwd が {repo} のセッションで打つ｡

## 横断ルール

- subagent の Edit は **Step 6 にのみ集約** ｡ Step 1〜5 の解析中は Edit 呼ばない (F3/F5 時の部分破損禁止)
- 前提条件の不足 (F1) は dispatch 前にエラー終了する｡ 対象の md へ触る前に止めるので、 部分的に添削された状態を作らない
- dispatch 後の失敗 (F2) は該当 step を skip して他の step は走る｡ 既にファイルへ手を入れた後なので、 途中で止めるより結果を返すほうが復旧しやすい
- F4 で部分 apply 済みになった場合、 「未 apply 分を Tier 2 化して親が適用する」 がデフォルトリカバリ
