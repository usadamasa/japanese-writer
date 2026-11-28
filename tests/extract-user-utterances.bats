#!/usr/bin/env bats
# writing-retrospective が同梱する利用者の発言の抽出フィルタのテスト
bats_require_minimum_version 1.5.0

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
FILTER="$REPO_ROOT/skills/writing-retrospective/extract-user-utterances.jq"

setup() {
  USER_SAYS='{"type":"user","sessionId":"s1","timestamp":"2026-11-01T00:00:00Z","message":{"role":"user","content":"結論を先に書いて"}}'
  USER_BLOCKS='{"type":"user","sessionId":"s1","timestamp":"2026-11-01T00:01:00Z","message":{"role":"user","content":[{"type":"text","text":"その節は要らない"}]}}'
  TOOL_RESULT='{"type":"user","sessionId":"s1","timestamp":"2026-11-01T00:02:00Z","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}'
  META='{"type":"user","isMeta":true,"sessionId":"s1","timestamp":"2026-11-01T00:03:00Z","message":{"role":"user","content":"skill の本文"}}'
  EDIT_MD='{"type":"assistant","sessionId":"s1","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/x/docs/a.md","old_string":"a","new_string":"b"}}]}}'
  EDIT_GO='{"type":"assistant","sessionId":"s1","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/x/main.go","old_string":"a","new_string":"b"}}]}}'
  SKILL_OWN='{"type":"assistant","sessionId":"s1","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"japanese-writer:proofread"}}]}}'
  SKILL_OTHER='{"type":"assistant","sessionId":"s1","message":{"content":[{"type":"tool_use","name":"Skill","input":{"skill":"other-plugin:writer"}}]}}'
}

@test ".md を編集したセッションから利用者の発言を取り出す" {
  run -0 jq -s -c -f "$FILTER" <<< "$(printf '%s\n' "$USER_SAYS" "$EDIT_MD")"
  [ "$(jq -r '.session_id' <<< "$output")" = "s1" ]
  [ "$(jq -r '.timestamp' <<< "$output")" = "2026-11-01T00:00:00Z" ]
  [ "$(jq -r '.text' <<< "$output")" = "結論を先に書いて" ]
}

@test "本文がブロックの配列なら text のブロックを取り出す" {
  run -0 jq -s -c -f "$FILTER" <<< "$(printf '%s\n' "$USER_BLOCKS" "$EDIT_MD")"
  [ "$(jq -r '.text' <<< "$output")" = "その節は要らない" ]
}

@test "tool_result と isMeta の user エントリは発言に数えない" {
  run -0 jq -s -c -f "$FILTER" <<< "$(printf '%s\n' "$USER_SAYS" "$TOOL_RESULT" "$META" "$EDIT_MD")"
  [ "$(wc -l <<< "$output" | tr -d ' ')" = "1" ]
  [ "$(jq -r '.text' <<< "$output")" = "結論を先に書いて" ]
}

@test "japanese-writer の skill を Skill ツールで呼んだセッションも対象にする" {
  run -0 jq -s -c -f "$FILTER" <<< "$(printf '%s\n' "$USER_SAYS" "$SKILL_OWN")"
  [ "$(jq -r '.text' <<< "$output")" = "結論を先に書いて" ]
}

@test "利用者が japanese-writer の slash command を打ったセッションも対象にする" {
  command='{"type":"user","sessionId":"s1","timestamp":"2026-11-01T00:04:00Z","message":{"role":"user","content":"<command-name>/japanese-writer:proofread</command-name>"}}'
  run -0 jq -s -c -f "$FILTER" <<< "$(printf '%s\n' "$USER_SAYS" "$command")"
  [ "$(jq -sr 'map(.text) | first' <<< "$output")" = "結論を先に書いて" ]
}

@test ".md を編集せず japanese-writer の skill も呼んでいないセッションは何も出さない" {
  run -0 jq -s -c -f "$FILTER" <<< "$(printf '%s\n' "$USER_SAYS" "$EDIT_GO" "$SKILL_OTHER")"
  [ -z "$output" ]
}
