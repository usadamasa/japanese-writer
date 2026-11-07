# crit のコメントを 1 件 1 行で取り出す。入力は transcript の 1 行か review.json。
# transcript ではコメントが approve 後 prompt の中に JSON 配列の 1 行として埋め込まれ、
# 埋め込まれる位置 (tool_result、Read の結果) が一定しないので、文字列を全部たどる。

def comment_fields:
  select(type == "object" and has("id") and has("body"))
  | {id, path, created_at, author, body, anchor, quote,
     replies: [.replies[]? | {author, body}]};

def review_file_comments:
  (.review_comments[]?),
  (.files // {} | to_entries[] | .key as $path | .value.comments[]? | . + {path: $path});

if type == "object" and (has("review_comments") or has("files")) then
  review_file_comments | comment_fields
else
  .. | strings | select(contains("\"replies\""))
  | split("\n")[] | select(startswith("[{"))
  | (try fromjson catch empty)
  | .[]? | comment_fields
end
