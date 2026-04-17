package slm

import "context"

// callChatCompletion はOpenAI互換APIを呼び出す（デフォルトパラメータ版）
func (c *Client) callChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return c.callChatCompletionWithParams(ctx, systemPrompt, userPrompt, 1024, 0.3)
}
