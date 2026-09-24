{{if eq .total_count 0}}Review approved with no comments — no changes requested.{{else}}Review approved. All comments are resolved.{{end}}
{{if gt .total_count 0}}
## 言い回しの指摘をルールへ移す

このレビューで付いたコメントの全件 (返信を含む):

{{.comments_json}}

まず `writing-feedback` skill が指すルールファイル (prh のルールセットと writing-gate の
`rules.json`) を置いているリポジトリを確認する。

- 今の cwd と同じリポジトリなら、この節の作業を subagent に任せる。`Agent` ツールで 1 本だけ
  起動し、結果を待たずに次節へ進む。
- 今の cwd と違うリポジトリなら、そのリポジトリを cwd にした別セッションへこの節の作業を
  引き継ぐ ([[cross-session-messaging]] または [[herdr-operations]] の手順に従う)。送信後は
  結果を待たずに次節へ進む。worktree での代用と `git -C` / `cd` での代用はしない。

いずれの場合も、渡す指示には次を含める。

- 上のコメント全件をそのまま渡す。
- 人間が読む文 (文書、コードコメント、ログやエラーのメッセージなど) の言い回しへの指摘 (語の選び方、語尾、くどさ、AI っぽさ、表記) を拾う。
  コードの命名・設計・事実誤りへの指摘は対象外。該当が無ければ何もせずに終える。
- 該当があれば `japanese-writer:writing-feedback` skill を invoke して、指摘ごとに登録の要否と置き場を判定する。
  crit のコメントは「言い回しを直された」に当たるので、その文書に固有の言い換え
  (固有名やその場の文脈に依存するもの) を除いて登録する。
  NG パターンと書き換え例の対は、コメント本文 (`body`) と返信 (`replies[].body`) から取る。
- 編集後はブランチを切って commit し、Draft PR まで作る。
{{end}}
## Approve 後の手順

レビューが通ったので、変更を commit して Draft PR まで用意する。以下を順に実行する。

Draft 解除とマージはここでは行わない。ユーザーの明示的な指示があるときだけ実行する。

### 1. 作業ブランチの用意

```bash
git status --short
git rev-parse --abbrev-ref HEAD
```

現在地と変更の有無で次の行動を振り分ける。

| 現在地 | 変更 | 対応 |
| --- | --- | --- |
| `main` / `master` | 未コミットの変更あり | ブランチを切って commit する |
| `main` / `master` | 変更なし | 「統合対象なし」と報告して終了する |
| フィーチャーブランチ | 未コミットの変更あり | そのまま commit する |
| フィーチャーブランチ | 変更なし かつ base との差分が 0 commit | 「統合対象なし」と報告して終了する |

```bash
git switch -c '<type>/<summary>'   # main / master に居るときだけ
git add -A
git commit -m '<type>(<scope>): <summary>'
```

- ブランチ名と commit メッセージは diff から生成する。ユーザーに確認しない
- ブランチ名は `<type>/<kebab-case-summary>`、commit メッセージは Conventional Commits 形式で、どちらも type を揃える
- `git add -A` はレビュー対象外の変更まで拾う。`git status --short` の内容が意図した差分だけか確認してから実行する

### 2. PR の有無を確認

```bash
gh pr view --json number,state,isDraft,mergeable,mergeStateStatus,url
```

PR が無ければ 3、あれば 4 へ。

### 3. PR が無い場合: Draft PR を作る

```bash
git push -u origin HEAD
gh pr create --draft --assignee @me --title '<title>' --body '<body>'
```

title は 70 文字未満、body は Summary と Test plan の 2 節。
作成した URL を報告して終了する。

### 4. PR がある場合: push して状態を報告する

```bash
git push
gh pr checks
```

PR の URL と checks の結果を報告して終了する。

| 状態 | 報告内容 |
| --- | --- |
| checks 全 pass (`no checks reported` も pass 扱い) | マージ可能であることを伝える |
| checks が pending | 残っている check 名 |
| checks が失敗 | 失敗した check のログ要約 |
| `mergeable: CONFLICTING` | コンフリクトしているファイル |

Draft 解除 (`gh pr ready`) とマージ (`gh pr merge`) は、ユーザーがそう指示したときだけ実行する。
