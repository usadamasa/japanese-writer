#!/usr/bin/env bats
# writing-retrospective が同梱する crit コメント抽出フィルタのテスト
bats_require_minimum_version 1.5.0

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
FILTER="$REPO_ROOT/skills/writing-retrospective/extract-crit-comments.jq"

setup() {
  COMMENTS='[{"scope":"line","path":"docs/a.md","id":"c_1","body":"結論を先に","anchor":"# 見出し","author":"Reviewer","replies":[{"id":"rp_1","body":"入れ替えた","author":"Claude Code"}]}]'
  PROMPT="$(printf 'approved: true\n\n## 言い回しの指摘をルールへ移す\n\n%s\n## Approve 後の手順\n' "$COMMENTS")"
}

@test "tool_result の文字列に埋め込まれたコメントを取り出す" {
  line="$(jq -cn --arg p "$PROMPT" '{type:"user",message:{content:[{type:"tool_result",content:$p}]}}')"
  run -0 jq -c -f "$FILTER" <<< "$line"
  [ "$(jq -r '.id' <<< "$output")" = "c_1" ]
  [ "$(jq -r '.path' <<< "$output")" = "docs/a.md" ]
  [ "$(jq -r '.body' <<< "$output")" = "結論を先に" ]
  [ "$(jq -r '.replies[0].body' <<< "$output")" = "入れ替えた" ]
}

@test "Read の結果のように深くネストした文字列からも取り出す" {
  line="$(jq -cn --arg p "$PROMPT" '{toolUseResult:{type:"text",file:{filePath:"/x/brief.md",content:$p}}}')"
  run -0 jq -c -f "$FILTER" <<< "$line"
  [ "$(jq -r '.id' <<< "$output")" = "c_1" ]
}

@test "review.json そのものからレビュー全体とファイルのコメントを取り出す" {
  review='{"review_comments":[{"id":"r_1","body":"全体として長い","scope":"review","author":"Reviewer","replies":[]}],"files":{"docs/b.md":{"comments":[{"id":"c_2","start_line":3,"body":"前提が抜けている","quote":"この方式","author":"Reviewer","replies":[]}]}}}'
  run -0 jq -c -f "$FILTER" <<< "$review"
  [ "$(wc -l <<< "$output" | tr -d ' ')" = "2" ]
  [ "$(jq -sr 'map(.id) | sort | join(",")' <<< "$output")" = "c_2,r_1" ]
  [ "$(jq -sr 'map(select(.id == "c_2"))[0] | .path + " " + .quote' <<< "$output")" = "docs/b.md この方式" ]
}

@test "コメントの JSON を含まない行と壊れた JSON は何も出さずに通す" {
  broken="$(jq -cn '{message:{content:"\"replies\" とだけ書いた文\n[{\"id\": 壊れている"}}')"
  run -0 jq -c -f "$FILTER" <<< "$broken"
  [ -z "$output" ]
}
