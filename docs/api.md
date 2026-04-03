# EHR Demo API仕様書

## 基本情報
- Base URL: `http://localhost:8080/api`
- Content-Type: `application/json`
- 認証: なし（デモ用）

## 共通レスポンス構造

### 成功時
```json
{
  "data": { ... },
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

### エラー時
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Patient not found"
  }
}
```

## エンドポイント一覧

### 患者 (Patients)

#### GET /api/patients
患者一覧（検索・ページネーション付き）

Query Parameters:
- `q` (string): 名前・カナ・MRNで検索
- `page` (int, default: 1)
- `limit` (int, default: 20)

#### GET /api/patients/:id
患者詳細

#### POST /api/patients
患者登録

#### PUT /api/patients/:id
患者更新

---

### 受診 (Encounters)

#### GET /api/patients/:id/encounters
受診履歴一覧

#### POST /api/patients/:id/encounters
受診登録

---

### SOAP記録

#### GET /api/encounters/:id/soap
SOAP記録取得

#### POST /api/encounters/:id/soap
SOAP記録作成

#### PUT /api/soap/:id
SOAP記録更新

---

### SLM連携

#### POST /api/slm/suggest/soap
問診テキストからSOAP提案を生成

Request:
```json
{
  "interview_text": "患者は3日前から頭痛を訴えている..."
}
```

Response:
```json
{
  "data": {
    "subjective": "3日前からの持続的な頭痛...",
    "objective": "バイタル: BP 130/85...",
    "assessment": "緊張性頭痛の疑い...",
    "plan": "鎮痛剤処方、1週間後再診..."
  },
  "meta": {
    "model": "qwen3.5-0.8b-medical",
    "is_mock": false,
    "latency_ms": 340
  }
}
```

#### POST /api/slm/suggest/summary
家族歴・社会歴の要約提案

Request:
```json
{
  "interview_text": "父親が糖尿病で...",
  "category": "family_history"
}
```

Response:
```json
{
  "data": {
    "suggestions": [
      {
        "field": "relation",
        "value": "父",
        "confidence": 0.95
      },
      {
        "field": "condition",
        "value": "2型糖尿病",
        "confidence": 0.90
      }
    ]
  },
  "meta": {
    "model": "qwen3.5-0.8b-medical",
    "is_mock": false,
    "latency_ms": 210
  }
}
```

#### GET /api/slm/health
SLM推論サーバーの接続状態確認

Response:
```json
{
  "data": {
    "status": "connected",
    "api_url": "http://localhost:8000",
    "model": "qwen3.5-0.8b-medical"
  }
}
```

`status` は `"connected"`（実SLM接続中）または `"mock_mode"`（モックフォールバック）。

---

### 既往歴 (Medical History)

#### GET /api/patients/:id/medical-history
患者の既往歴一覧

#### POST /api/patients/:id/medical-history
既往歴登録

Request:
```json
{
  "condition": "高血圧症",
  "onset_date": "2020-05-01",
  "status": "治療中",
  "notes": "ACE阻害薬にて管理中"
}
```

#### PUT /api/medical-history/:id
既往歴更新

#### DELETE /api/medical-history/:id
既往歴削除

---

### 家族歴 (Family History)

#### GET /api/patients/:id/family-history
患者の家族歴一覧

#### POST /api/patients/:id/family-history
家族歴登録

Request:
```json
{
  "relation": "父",
  "condition": "2型糖尿病",
  "notes": "60歳で発症"
}
```

#### PUT /api/family-history/:id
家族歴更新

#### DELETE /api/family-history/:id
家族歴削除

---

### 社会歴 (Social History)

#### GET /api/patients/:id/social-history
患者の社会歴一覧

#### POST /api/patients/:id/social-history
社会歴登録

Request:
```json
{
  "category": "喫煙",
  "description": "20本/日×30年",
  "notes": "禁煙指導中"
}
```

#### PUT /api/social-history/:id
社会歴更新

#### DELETE /api/social-history/:id
社会歴削除

---

### 問診記録 (Interview Notes)

#### GET /api/encounters/:id/interviews
受診に紐づく問診記録一覧

#### POST /api/encounters/:id/interviews
問診記録保存

Request:
```json
{
  "raw_text": "患者は3日前から頭痛を訴えている。市販の鎮痛剤を服用したが改善しない。",
  "structured_data": null
}
```

---

### インポート

#### POST /api/import/patients/csv
CSVファイルから患者データをインポート

Content-Type: multipart/form-data

---

### 開発用

#### GET /api/seed
ダミーデータを生成・投入（開発環境のみ）
