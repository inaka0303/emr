# Claude Code 初期セットアップ & プロンプトガイド

## 前提条件

1. Claude Code がインストール済み（`npm install -g @anthropic-ai/claude-code`）
2. Max 5x プランでログイン済み
3. Node.js 18+ インストール済み
4. Go 1.22+ インストール済み

## Step 1: Agent Teams を有効化

Claude Codeの設定ファイルに以下を追加:

```bash
# Claude Code内で実行
claude config set env.CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS 1

# または settings.json を直接編集
# ~/.claude/settings.json に以下を追加:
```

```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  }
}
```

設定後、Claude Codeを再起動してください。

## Step 2: プロジェクトをセットアップ

```bash
# プロジェクトディレクトリに移動（CLAUDE.mdがある場所）
cd ehr-demo

# Gitリポジトリを初期化
git init
git add .
git commit -m "init: プロジェクト初期セットアップ"
```

## Step 3: Claude Code を起動

```bash
claude
```

## Step 4: Phase 1 起動プロンプト

以下をClaude Codeに貼り付けてください:

---

```
CLAUDE.mdを読んで、EHR（電子カルテ）デモプロジェクトの概要を把握してください。
このプロジェクトではAgent Teamsを使って並列開発を行います。
あなたはPM（チームリーダー）として、.claude/agents/pm-lead.md の定義に従って全体を取りまとめてください。

## Phase 1: 基盤構築

以下のチームメイトをspawnして、Phase 1のタスクを並列で進めてください。

### チームメイト1: frontend-dev
.claude/agents/frontend-dev.md の定義に従う。
最初のタスク:
- frontend/ の Vite + React + TypeScript + Tailwind プロジェクト初期化
- 共通レイアウトコンポーネントの作成（PC: サイドバー+メイン+右パネル、スマホ: ボトムナビ）
- 患者一覧画面のモックUI（APIはまだないのでダミーデータで）

### チームメイト2: backend-dev
.claude/agents/backend-dev.md の定義に従う。
最初のタスク:
- backend/ の Go + Echo プロジェクト初期化
- SQLiteのDBスキーマ作成（CLAUDE.mdのデータモデルに従う）
- 患者CRUD APIの実装（GET /api/patients, GET /api/patients/:id, POST /api/patients）
- ダミーデータのシード機能

### チームメイト3: qa-reviewer
.claude/agents/qa-reviewer.md の定義に従う。
最初のタスク:
- frontend-devとbackend-devの初期実装が完了したらレビュー
- API型定義（frontend/src/types/api.ts）とGoの構造体の整合性チェック
- docs/api.md と実際のAPIレスポンスの一致確認

frontend-devとbackend-devは並列で進められます。qa-reviewerは両者の初期実装が揃ってからレビューに入ってください。
Phase 1の完了時に、進捗報告をお願いします。
```

---

## Step 5: Phase 2 以降のプロンプト

Phase 1 の完了報告を受けたら、以下で Phase 2 を開始:

```
Phase 1の成果を確認しました。Phase 2（カルテ機能）に進みましょう。

### frontend-dev のタスク:
- SOAP形式カルテ入力UI（S/O/A/P のセクション分けエディタ）
- カルテ履歴表示（時系列リスト）
- 患者詳細画面（既往歴、家族歴、社会歴のタブ表示）
- PC/スマホのレスポンシブ対応を確認

### backend-dev のタスク:
- 受診(encounters) CRUD API
- SOAP記録 CRUD API
- 既往歴・家族歴・社会歴の CRUD API
- 問診記録の保存API

### qa-reviewer のタスク:
- カルテ入力UIのキーボード操作テスト
- レスポンシブレイアウトのチェック
- APIの入出力テスト

Phase 2完了時に報告をお願いします。
```

Phase 3（SLM連携）:

```
Phase 3（SLM連携）に進みましょう。これがプロジェクトの核心です。

### backend-dev のタスク:
- SLMクライアント実装（internal/slm/client.go）
  - OpenAI互換 /v1/chat/completions を呼ぶHTTPクライアント
  - 環境変数 SLM_API_URL で推論サーバーURLを設定
  - 接続失敗時のモックフォールバック
- モック実装（internal/slm/mock.go）
  - テンプレートベースで問診テキストに応じたSOAP提案を返す
  - 0.5-1秒のdelay付き
- /api/slm/suggest/soap と /api/slm/suggest/summary エンドポイント
- /api/slm/health でSLM接続状態確認

### frontend-dev のタスク:
- 問診入力画面の実装
  - 自由記述テキストエリア
  - 「SLMに提案を依頼」ボタン
- サジェストUI（最重要！）
  - SOAP各フィールドにグレーのインライン提案を表示
  - Tab で accept、Esc で dismiss、→ で次のフィールド
  - accept後は通常テキストに変わり編集可能
  - ローディング中のスケルトンアニメーション
- 家族歴・社会歴の要約提案UI
  - suggestions配列をリスト表示
  - 各提案にaccept/dismissボタン

### qa-reviewer のタスク:
- SLMサジェストUIの操作テスト（Tab/Esc/→の動作）
- モックレスポンスの内容が医療的に不自然でないかチェック
- エラーケース（API timeout、空レスポンス）のUI確認

このPhaseの品質がデモの印象を決めます。特にサジェストUIの操作感に注力してください。
```

## トラブルシューティング

### Agent Teamsが起動しない
- `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` が設定されているか確認
- Claude Codeを完全に再起動（`exit` → `claude`）
- Opus 4.6モデルが使えているか確認

### チームメイトが応答しない
- `チームメイトの状態を確認して` とPMに聞く
- 特定のチームメイトに直接話しかけることも可能（分割ペインで）

### コンテキストが長くなりすぎたら
- `compact` コマンドでコンテキストを圧縮
- 新しいセッションを開始し、前の成果をgitコミットから復元

### Agent Teamsを使わずに進めたい場合
Agent Teamsがうまく動かない場合、サブエージェント方式でも進められます:
```
CLAUDE.mdを読んで、Phase 1の基盤構築を進めてください。
まずbackendのプロジェクト初期化とDB作成、
次にfrontendの初期化とレイアウト作成、
最後にAPI接続テストの順で進めてください。
```
この場合は1セッションで順番に処理するので、並列性はないが安定性は高いです。
