#!/bin/bash
set -euo pipefail
# hook-logger.sh
# hook スクリプト共通のログ・dry-run ヘルパー
#
# 使い方:
#   HOOK_NAME="guard-broad-wildcard"
#   source "$SCRIPT_DIR/lib/hook-logger.sh"
#
# 環境変数:
#   DRY_RUN=1  全操作をシミュレーション表示し、実際の副作用を起こさない
#   HOOK_NAME  ログプレフィックスに使用するスクリプト名

DRY_RUN="${DRY_RUN:-0}"
HOOK_NAME="${HOOK_NAME:-hook}"

# DRY_RUN=1 かどうかを判定
is_dry_run() {
  [ "$DRY_RUN" = "1" ]
}

# ログプレフィックスは source 時に一度だけ確定させる。
# 各ログ出力で $(_log_prefix) を呼ぶとサブシェルを毎回フォークするため、
# 確定値を変数に格納して参照する (DRY_RUN/HOOK_NAME は source 前に設定される前提)。
# dry-run: "[DRY-RUN]", 通常: "[$HOOK_NAME]"
if is_dry_run; then
  _LOG_PREFIX="[DRY-RUN]"
else
  _LOG_PREFIX="[$HOOK_NAME]"
fi

# 後方互換・テスト用: 確定済みプレフィックスを返す
_log_prefix() {
  printf '%s\n' "$_LOG_PREFIX"
}

# mkdir -p + ログ出力。dry-run 時は mkdir しない
logged_mkdir() {
  local target="$1"
  if is_dry_run; then
    echo "$_LOG_PREFIX MKDIR  $target" >&2
  else
    echo "$_LOG_PREFIX MKDIR  $target" >&2
    mkdir -p "$target"
  fi
}

# コマンド実行 + ログ出力。dry-run 時は実行しない
logged_cmd() {
  if is_dry_run; then
    echo "$_LOG_PREFIX CMD    $*" >&2
  else
    echo "$_LOG_PREFIX CMD    $*" >&2
    "$@"
  fi
}

# スキップ理由の通知
log_skip() {
  local reason="$1"
  echo "$_LOG_PREFIX SKIP   ($reason)" >&2
}

# 情報メッセージ (常に出力)
log_info() {
  local msg="$1"
  echo "$_LOG_PREFIX INFO   $msg" >&2
}

# エラーメッセージ (常に出力、重大度: ERROR)
log_error() {
  local msg="$1"
  echo "$_LOG_PREFIX ERROR  $msg" >&2
}

# PreToolUse フックで deny を返す。jq で JSON エスケープするため安全。
deny_tool() {
  local reason="$1"
  jq -cn --arg r "$reason" \
    '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":$r}}'
}

# PreToolUse フックで ask (プロンプト) を返す。jq で JSON エスケープするため安全。
ask_tool() {
  local reason="$1"
  jq -cn --arg r "$reason" \
    '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":$r}}'
}
