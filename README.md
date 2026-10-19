# EMR / EHR Demo

医療用SLMによる記載支援を検証する電子カルテデモ。患者・受診・問診の管理、SOAPの生成と編集、入院時サマリ、RAGの根拠表示、AIあり／なしの比較実験と結果出力を含みます。

## 構成

| 場所 | 役割 |
| --- | --- |
| `frontend/` | React 19・TypeScript・Vite 8・Tailwind CSS 3による日本語UI |
| `backend/cmd/server/main.go` | Echoの起動、依存関係の組み立て、APIルート、環境変数 |
| `backend/internal/{handler,service,repository,model}/` | HTTP処理・業務処理・SQLite操作・データ型 |
| `backend/internal/slm/` | 推論API、LoRA切り替え、ストリーミング、モック、RAG連携 |
| `backend/db/migrations/` | 起動時に適用するSQL。履歴を含めて保持する |
| `backend/db/seed/` | 8症例・学生32試行（A01–A32）とデモ患者の投入 |
| `scripts/` | 実験準備、ウォームアップ、結果出力、参照回答作成、採点 |
| `references/` | 実験用の固定参照回答と採点手順 |
| `generate_admission_summary.py` | 入院時サマリのSFT学習データ30件を生成する独立スクリプト |
| `docs/api.md` | 基本APIの説明。追加機能を含む現行ルートはサーバー実装を参照 |
| `.claude/`・`CLAUDE.md` | 開発エージェントの役割と開発規約 |
| `exports/` | ローカルの実験出力・過去資料ZIP（Git管理外） |

ブラウザは `/api` を介してGoへ接続します。開発時はViteがAPIを転送し、GoがSQLiteおよび外部のSLM・RAGサーバーと通信します。モデル本体、推論サーバー、RAGサーバーはこのリポジトリには含まれません。

画面の入口は `frontend/src/components/common/Layout.tsx` です。SOAP生成は `useSoapDraftCache.ts` とバックエンドの `soap_draft.go` でキャッシュ・ストリーミングを扱い、保存済みSOAPとは別に管理します。通常画面はAPI障害時にデモ用モック患者を表示でき、`?attempt=A01` などの実験画面では試行・患者を固定します。

## 開発起動

必要環境はGo 1.25以上（`backend/go.mod`）、Node.js 20.19以上の20系または22.12以上、npmです。PythonスクリプトにはPython 3.10以上を使用します。

バックエンドを起動します。

```bash
cd backend
go run ./cmd/server
```

別ターミナルでフロントエンドを起動します。

```bash
cd frontend
npm ci
npm run dev
```

通常は `http://localhost:5173` を開きます。バックエンドの稼働確認は `http://localhost:8080/health` です。SQLiteは**サーバーの作業ディレクトリ**の `ehr-demo.db` に作成されるため、通常は `backend/` から起動してください。

| 環境変数 | 既定値・用途 |
| --- | --- |
| `PORT` | Goの待受ポート。既定は `8080` |
| `SLM_API_URL` | 主推論サーバー。既定は `http://localhost:8000`。未接続時はモック |
| `SLM_ADMISSION_URL` | 入院時サマリ専用サーバー。未指定時は主サーバーを使用 |
| `RAG_API_URL` | 既定は `http://localhost:8082`。空文字指定で生成時の自動注入を無効化。ただし手動検索APIは既定URLを使用 |
| `ENABLE_CROSS_ENCOUNTER_SUMMARY` | `false` で過去受診要約の付与を無効化 |
| `APP_ENV` | `production` でシード・テスト患者リセットAPIを無効化 |
| `ALLOWED_ORIGINS` | CORS許可元をカンマ区切りで指定。既定はlocalhostの5173・5174 |
| `VITE_API_PROXY_TARGET` | ViteのAPI転送先。既定は `http://localhost:8080` |
| `ACI_CASE_DIR` | シード時に読み込む外部症例JSONのディレクトリ |

既存DBには実験記録が含まれ得ます。`GET /api/seed` は既存の患者・SOAP・試行などを削除して実験データを作り直すため、既存DBの起動確認には使用しないでください。通常画面の読み込み時には、デモ患者 `MRN-0021` の問診・ドラフト・入院時サマリをリセットする処理があります。

## 実験・採点スクリプト

| スクリプト | 用途 |
| --- | --- |
| `prepare_resident_experiment.py` | 研修医用の独立試行R01–R16をDBに追加 |
| `check_experiment_ready.sh` | 試行の開始前状態をAPIで確認 |
| `warmup_experiment.sh` | 推論をウォームアップ |
| `export_experiment_results.py` | DBから実験結果をExcel・CSVへ出力 |
| `build_experiment_references.py` | Google Docsまたはローカルテキストから参照回答を生成 |
| `curate_experiment_references.py` | 参照回答の採点用key_factsを整備 |
| `validate_experiment_references.py` | 固定参照回答の整合性を検証 |
| `score_experiment_results.py` | 薬剤・バイタル・診断などとテキスト指標を採点 |
| `build_judge_prompts.py` | 手動LLM評価のプロンプトを作成 |
| `merge_judge_results.py` | 手動LLM評価を採点Excelに統合 |
| `export_current_experiment.sh` | 結果出力・採点・評価プロンプト作成をまとめて実行 |

詳細は [references/README.md](references/README.md) を参照してください。各スクリプトは役割が異なるため、似た名称でも重複ファイルではありません。

外部環境への依存は以下のとおりです。

- シードは外部の `data/aci_jp_cardio/outpatient/cases/` にある症例JSONを必要とします。必要に応じて `ACI_CASE_DIR` を設定します。
- 採点はプロジェクトの隣の `data/aci_jp_cardio/admission/eval_runner/` にある `metrics_drug.py`・`metrics_text.py`・`metrics_vitals.py` を読み込みます。このチェックアウトには含まれません。
- 採点のテキスト指標用パッケージは `requirements-scoring.txt`、Excel出力には別途 `openpyxl` が必要です。
- 一部スクリプトの既定パスは実験サーバー用の `/home/junkanki/naka/…` や `/tmp/emr-exp-run/…` です。ローカル実行時は `--db`・`--output` など、まとめて出力する場合は `EXPORT_DIR` とDB引数を指定します。
- `generate_admission_summary.py` は固有の学習用サンプルを含み、出力先が `/data2/junkanki/naka/data/sft_admission_summary.jsonl` に固定されています。通常のアプリ起動では実行しません。

シードテストが必要とする症例ファイルは `JC-AMI-A.json`・`JC-AMI-B.json`・`JC-PE-A.json`・`JC-PE-B.json`・`JC-AD-A.json`・`JC-AD-B.json`・`JC-AHF-A.json`・`JC-AHF-B.json` の8件です。テストはリポジトリの隣の `data/aci_jp_cardio/outpatient/cases/` を直接確認します（テスト内で `ACI_CASE_DIR` を設定）。2026-10-19の作業環境には、この症例ディレクトリと採点用の `eval_runner/` が整理前からありませんでした。`references/experiment_references.jsonl` は採点の固定参照回答であり、シード入力JSONや評価コードの代わりにはなりません。

## 検証

```bash
(cd frontend && npm run build)
(cd frontend && npm run lint -- --max-warnings 0)
(cd frontend && npm test)
(cd backend && go test -v ./...)
python3 scripts/validate_experiment_references.py
```

ブラウザテストの初回実行前に、`frontend/` で `npx playwright install chromium` を実行してください。`npm test` はテストの型チェックとPlaywrightによる7件の回帰テストを実行します。APIはテスト内で置き換えるため、実験DBや推論サーバーは使用しません。

2026-10-19の不要ファイル整理時点で、生成JS・CSSの整理前とのバイト単位の一致とPython 9ファイル・Shell 3ファイルの構文を確認しました。その後、残っていたLintの5エラー・1警告を修正し、Lint 0件、ビルド成功、ブラウザ回帰テスト7件成功、Goテスト成功（サブテストを含む26件）、固定参照回答8件の検証成功を確認しています。

SOAPの編集状態は受診・実験のリセット・保存済み記録の更新単位で初期化し、生成表示は受信中のドラフトから導出します。採用・手動編集済みのセクションは生成の更新から保護し、保存成功時にはブラウザの復元用ドラフトを消します。問診・入院時サマリでは、切り替え前の入力に対する遅い応答を無視します。

シードのGoテスト1件は外部症例データがないためスキップされました。SLM・RAGの実サーバー接続と、外部の評価モジュールを使う採点全体は未検証です。

## 2026-10-19のファイル整理

- 未参照の旧SOAP提案フック・タイピング表示フック、Goの未使用互換ラッパー、未使用のフロントエンドAPIラッパーを削除。
- Vite/React雛形由来の未使用画像3件、未使用の `public/icons.svg`、雛形README、初期開発用の `STARTUP_GUIDE.md` を削除。起動手順は本READMEへ集約。
- Goプロジェクトに誤生成された空の `backend/package-lock.json`、`.DS_Store`、再生成可能な `frontend/dist/` を削除。
- `exports/` 内の以下2フォルダは全62ファイルについてZIPと内容が一致することを確認し、展開済みフォルダを削除。ZIPはそのまま保持。
  - `codex_geniac_execution_package_20260702_183616_af068cc`
  - `fable5_geniac_project_package_20260702_174425_af068cc`

SQLiteのDB・WAL・SHM、固定参照回答、学習データ生成スクリプト、抄録原稿、企画資料ZIP、現在使用中のモック・マイグレーション・依存パッケージは保持しています。原稿や企画資料は固有の内容があり、日付だけでは不要と判断できません。Git管理されていた削除ファイルの元の内容は、整理前のコミット `af068cc` で確認できます。
