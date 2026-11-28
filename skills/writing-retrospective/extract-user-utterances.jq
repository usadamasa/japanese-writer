# 利用者の発言を 1 件 1 行で取り出す。入力は transcript 1 本を -s で読んだ配列。
# 文章を扱ったセッション (.md を編集したか、japanese-writer の skill を呼んだ) だけを対象にする。
# 本文は文字列とブロックの配列の両方があるので、配列なら text のブロックだけをつなぐ。

def text_of:
  if type == "string" then .
  else [.[]? | select(.type == "text") | .text] | join("\n")
  end;

def tool_uses: .message.content | arrays | .[] | select(.type == "tool_use");

def utterances:
  .[] | select(.type == "user" and (.isMeta | not))
  | {session_id: .sessionId, timestamp, text: (.message.content | text_of)}
  | select(.text != "");

def writes_md:
  any(.[] | tool_uses; (.input.file_path? // "") | endswith(".md"));

def uses_own_skill:
  any(.[] | tool_uses; .name == "Skill" and ((.input.skill? // "") | startswith("japanese-writer:")))
  or any(utterances; .text | contains("<command-name>/japanese-writer:"));

if writes_md or uses_own_skill then utterances else empty end
