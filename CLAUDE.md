# EHR Demo 開発ガイド

医療用SLMによる電子カルテ記載支援のデモ・比較実験プロジェクト。
現在の構成、起動手順、外部依存、実験スクリプトの一覧は [README.md](README.md) を参照する。

## 技術スタックと構成

- フロントエンド: React 19 / TypeScript / Tailwind CSS 3 / Vite 8。
- バックエンド: Go 1.25以上 / Echo / SQLite（modernc.org/sqlite、CGO不要）。
- UIの状態管理: Reactのhooks。日本語UI、mobile-first。
- 推論: Goから外部のOpenAI互換APIを呼ぶ。LoRA切り替え・専用入院サマリサーバー・RAG連携があり、未接続時のモックを含む。
- 実験: 8症例、AIあり／なしの独立attempt、SOAP保存、操作・時間の記録、Pythonによる出力・採点。

```text
frontend/src/
  components/{common,patient,chart,slm}/
  pages/ hooks/ types/ api/ utils/
backend/
  cmd/server/main.go
  internal/{handler,model,repository,service,slm}/
  db/{migrations,seed}/
scripts/
references/
docs/api.md
.claude/{pm-lead,frontend-dev,backend-dev,qa-reviewer}.md
```

## コーディング規約

### フロントエンド

- 関数コンポーネントとhooksを使う。
- propsはinterfaceで型定義する。
- コンポーネント名はPascalCase、hooks/utilsはcamelCase。
- Tailwindのユーティリティを優先し、必要時のみカスタムCSSを使う。
- `frontend/src/types/api.ts` のAPI型をGoの構造体と一致させる。
- 日本語コメントを使用してよい。

### バックエンド

- Go標準のディレクトリレイアウトとhandler / service / repositoryの役割を維持する。
- 標準的なGoのerror wrappingを使う。
- APIレスポンスは共通のJSON構造を使う。
- ログはslog、テストはテーブル駆動を基本とする。

### Git・共同作業

- conventional commits（feat:, fix:, docs:, refactor:など）を使用する。
- ブランチはmain / develop / feature/*、コミットは小さく1機能単位にする。
- チーム作業時のファイル所有権: フロント担当は `frontend/` のみ、バック担当は `backend/` のみを編集する。
- 同じファイルを同時編集しない。`docs/api.md` の変更はPMに報告し、共有ファイル（`docs/api.md`・`CLAUDE.md`）の編集はPM経由にする。
- 各フェーズの完了時、PMは進捗を報告し、次フェーズを確認する。役割定義は `.claude/` 直下にある。

## データと検証

- DB・WAL・SHM、実験出力、固定参照回答を一時生成物として削除しない。
- シードAPIは既存データを消して作り直す。起動・動作確認には既存の実験DBを使ってリセットしない。
- フロントの確認は `npm run build` と `npm run lint`、バックは `go test -v ./...`。
- 外部症例データがない環境ではシードテストがスキップされる。採点にも外部Pythonモジュールが必要。
- 既存のLint違反など、検証できなかった項目や制約は結果に明記する。
