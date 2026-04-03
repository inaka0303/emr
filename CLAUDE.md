# EHR Demo - 電子カルテデモアプリケーション

## プロジェクト概要

学会発表用の電子カルテ（EHR）デモアプリケーション。
自社開発の医療用SLM（Small Language Model）の活用例として、
問診データからSOAP形式の記述提案・家族歴/社会歴の要約提案を行う機能をデモする。

**最優先ゴール**: SLMの推論能力を学会でインパクトをもって見せること
**開発期限**: 2〜3ヶ月後の学会デモ

## 技術スタック

| レイヤー | 技術 | 備考 |
|---|---|---|
| フロントエンド | React 18+ / TypeScript / Tailwind CSS | Viteでビルド |
| バックエンド | Go (1.22+) | Echo or net/http |
| DB | SQLite | modernc.org/sqlite (CGO不要) |
| SLM連携 | OpenAI互換REST API | プレースホルダーで先行実装 |
| 状態管理 | Zustand or React Context | 軽量に保つ |

## ディレクトリ構成

```
ehr-demo/
├── CLAUDE.md                    # このファイル
├── .claude/
│   └── agents/                  # エージェント定義
├── frontend/
│   ├── src/
│   │   ├── components/          # UIコンポーネント
│   │   │   ├── common/          # 汎用コンポーネント
│   │   │   ├── patient/         # 患者関連
│   │   │   ├── chart/           # カルテ関連
│   │   │   └── slm/             # SLM連携UI
│   │   ├── pages/               # ページコンポーネント
│   │   ├── hooks/               # カスタムフック
│   │   ├── types/               # TypeScript型定義
│   │   ├── api/                 # APIクライアント
│   │   └── utils/               # ユーティリティ
│   ├── package.json
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   └── vite.config.ts
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # エントリポイント
│   ├── internal/
│   │   ├── handler/             # HTTPハンドラ
│   │   ├── model/               # データモデル
│   │   ├── repository/          # DB操作
│   │   ├── service/             # ビジネスロジック
│   │   └── slm/                 # SLMクライアント
│   ├── db/
│   │   ├── migrations/          # マイグレーション
│   │   └── seed/                # ダミーデータ生成
│   └── go.mod
├── docs/
│   └── api.md                   # API仕様書
└── scripts/
    ├── seed.go                  # データシード
    └── dev.sh                   # 開発用起動スクリプト
```

## データモデル（コア）

### patients（患者）
- id, mrn (Medical Record Number), name, name_kana, birth_date
- gender, blood_type, phone, address
- emergency_contact_name, emergency_contact_phone
- created_at, updated_at

### encounters（受診）
- id, patient_id, encounter_date, encounter_type (外来/入院/救急)
- department, attending_doctor, status (進行中/完了)
- chief_complaint (主訴)
- created_at, updated_at

### soap_notes（SOAP記録）
- id, encounter_id, author
- subjective, objective, assessment, plan
- is_slm_suggested (SLM提案フラグ)
- created_at, updated_at

### medical_history（既往歴）
- id, patient_id, condition, onset_date, status, notes

### family_history（家族歴）
- id, patient_id, relation, condition, notes
- is_slm_suggested

### social_history（社会歴）
- id, patient_id, category (喫煙/飲酒/職業/運動 等)
- description, notes
- is_slm_suggested

### interview_notes（問診記録 - SLM入力用）
- id, encounter_id, raw_text (自由記述テキスト)
- structured_data (JSON - SLM処理結果)
- created_at

## 画面構成

### PC版レイアウト
- 左サイドバー: 患者リスト + 検索
- メインエリア: タブ切り替え (カルテ / 問診 / 履歴 / 患者情報)
- 右パネル: SLM提案エリア (問診時に表示)

### スマホ版レイアウト
- ボトムナビゲーション: 患者一覧 / カルテ / 問診 / 設定
- フルスクリーン切り替え
- SLM提案はモーダルまたはインラインで表示

### 主要画面
1. **患者一覧** - 検索・フィルタ付きリスト
2. **患者詳細** - 基本情報、既往歴、家族歴、社会歴のタブ表示
3. **カルテ入力** - SOAP形式のエディタ、過去のカルテ履歴表示
4. **問診入力** - 自由記述テキストエリア + SLMサジェストUI
5. **SLM提案ビュー** - Tab/Enterでaccept、→で次の提案へ

## SLM連携仕様

### モデル情報
- **モデル**: Qwen3.5 0.8B (ファインチューニング済み医療用SLM)
- **推論**: ローカル実行（同一マシンまたはLAN内）
- **推論サーバー**: vLLM / Ollama / llama.cpp のいずれか（OpenAI互換API）

### アーキテクチャ
```
┌──────────────┐    HTTP     ┌──────────────┐    HTTP     ┌──────────────────┐
│  React       │ ◄────────► │  Go Backend  │ ◄────────► │  推論サーバー      │
│  Frontend    │   :5173     │  :8080       │   :8000     │  (vLLM/Ollama)   │
│              │             │              │             │  Qwen3.5 0.8B    │
└──────────────┘             └──────────────┘             └──────────────────┘
```
- 電子カルテ(Go)が推論サーバーのOpenAI互換APIを呼ぶ構成
- 推論サーバーのURLは環境変数 `SLM_API_URL` で設定（デフォルト: http://localhost:8000）
- SLMが未起動の場合はGoバックエンド内のモックにフォールバック

### Goバックエンド → 推論サーバー通信

GoバックエンドがSLMを呼ぶ際は、OpenAI互換の `/v1/chat/completions` を使う:

```go
// backend/internal/slm/client.go
// 推論サーバーへのリクエスト（OpenAI互換）
POST ${SLM_API_URL}/v1/chat/completions
{
  "model": "qwen3.5-0.8b-medical",
  "messages": [
    {"role": "system", "content": "あなたは医療記録の作成を支援するAIです..."},
    {"role": "user", "content": "以下の問診内容からSOAP形式の記録を提案してください:\n\n{問診テキスト}"}
  ],
  "temperature": 0.3,
  "max_tokens": 1024
}
```

### フロントエンド → Goバックエンド API

フロントから直接推論サーバーは叩かない。Go経由で:

```
POST /api/slm/suggest/soap
  Request:  { "interview_text": "患者は3日前から頭痛..." }
  Response: {
    "data": {
      "subjective": "3日前からの持続的な頭痛...",
      "objective": "バイタル: BP 130/85...",
      "assessment": "緊張性頭痛の疑い...",
      "plan": "鎮痛剤処方、1週間後再診..."
    },
    "meta": { "model": "qwen3.5-0.8b-medical", "is_mock": false, "latency_ms": 340 }
  }

POST /api/slm/suggest/summary
  Request:  { "interview_text": "...", "category": "family_history|social_history" }
  Response: {
    "data": {
      "suggestions": [
        { "field": "relation", "value": "父", "confidence": 0.95 },
        { "field": "condition", "value": "2型糖尿病", "confidence": 0.90 }
      ]
    },
    "meta": { "model": "qwen3.5-0.8b-medical", "is_mock": false }
  }
```

### モック（フォールバック）実装
- 推論サーバーに接続できない場合、Go内のモックが応答
- モック時は `meta.is_mock: true` をセット
- モックは0.5-1秒のdelayを入れてリアルな体感にする
- テンプレートベースで、入力テキストのキーワードに応じた応答を返す
- **開発初期はモックで進め、SLM完成後にURLを差し替えるだけ**

### サジェストUI仕様
- 問診テキスト入力後、SLMが提案を生成
- 提案はグレーのインラインテキストとして表示（GitHub Copilotライク）
- `Tab` キーで現在の提案をaccept
- `Esc` で提案を却下
- `→` キーで次の提案フィールドへ移動
- accept済みの項目は通常テキストに変わり、編集可能

### デモ時の構成
学会デモでは以下の構成を想定:
- ノートPC1台で完結（Go + 推論サーバー + ブラウザ）
- 0.8BモデルならCPU推論でも数秒以内にレスポンス可能
- GPU搭載ならさらに高速（デモの印象が良い）
- 事前にモデルをダウンロード・変換しておく

## コーディング規約

### フロントエンド
- コンポーネントは関数コンポーネント + hooks
- propsの型はinterfaceで定義（typeよりinterface優先）
- ファイル名: PascalCase (コンポーネント), camelCase (hooks/utils)
- CSS: Tailwind CSS ユーティリティ優先、必要時のみカスタムCSS
- レスポンシブ: mobile-first (`sm:`, `md:`, `lg:` のブレークポイント)
- 日本語UI、コメントは日本語OK

### バックエンド
- Go標準のディレクトリレイアウト
- エラーハンドリング: 標準的なGo error wrapping
- APIレスポンス: JSON、統一的なレスポンス構造体
- ログ: slog パッケージ
- テスト: テーブル駆動テスト

### 共通
- Git: conventional commits (feat:, fix:, docs:, refactor:)
- ブランチ: main, develop, feature/*
- コミットは小さく、1機能1コミット

## 開発フェーズ

### Phase 1: 基盤構築（Week 1-2）
- [ ] プロジェクト初期化 (Vite + Go mod)
- [ ] DBスキーマ・マイグレーション
- [ ] 基本APIエンドポイント (CRUD)
- [ ] 共通UIコンポーネント (レイアウト、ナビ、テーブル)
- [ ] 患者一覧・検索画面

### Phase 2: カルテ機能（Week 3-4）
- [ ] SOAP形式カルテ入力UI
- [ ] カルテ履歴表示
- [ ] 患者詳細画面 (既往歴、家族歴、社会歴)
- [ ] レスポンシブ対応

### Phase 3: SLM連携（Week 5-6）
- [ ] SLMプレースホルダーAPI
- [ ] 問診入力UI
- [ ] サジェストUI (Tab accept)
- [ ] SOAP自動提案フロー
- [ ] 家族歴・社会歴の要約提案

### Phase 4: デモ仕上げ（Week 7-8）
- [ ] ダミーデータ充実
- [ ] CSVインポート機能
- [ ] デモシナリオ用のデータ準備
- [ ] UIポリッシュ
- [ ] 学会デモ用のウォークスルー整備

## エージェント間ルール

1. **ファイル所有権**: フロントは `frontend/` のみ、バックは `backend/` のみを編集
2. **API契約**: `docs/api.md` を変更する場合は必ずPMに報告
3. **型の共有**: APIレスポンスの型は `frontend/src/types/api.ts` に定義、バックエンドの構造体と一致させる
4. **コンフリクト回避**: 同じファイルを同時編集しない。共有ファイル（docs/api.md, CLAUDE.md）の編集はPM経由
5. **報告**: 各フェーズの完了時、PMは人間に進捗報告と次フェーズの確認を行う
