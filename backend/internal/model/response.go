package model

// APIResponse は統一的なAPIレスポンス構造体
type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error *APIError   `json:"error,omitempty"`
	Meta  *Meta       `json:"meta,omitempty"`
}

// APIError はエラー情報を表す構造体
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta はページネーション等のメタ情報を表す構造体
type Meta struct {
	Total int `json:"total,omitempty"`
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
}

// NewSuccessResponse は成功レスポンスを生成する
func NewSuccessResponse(data interface{}, meta *Meta) APIResponse {
	return APIResponse{
		Data: data,
		Meta: meta,
	}
}

// NewErrorResponse はエラーレスポンスを生成する
func NewErrorResponse(code, message string) APIResponse {
	return APIResponse{
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}

// SLMMeta はSLMレスポンスのメタ情報
type SLMMeta struct {
	Model     string `json:"model"`
	IsMock    bool   `json:"is_mock"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// SLMResponse はSLM APIの統一レスポンス
type SLMResponse struct {
	Data interface{} `json:"data"`
	Meta SLMMeta     `json:"meta"`
}
