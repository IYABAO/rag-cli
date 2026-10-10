package main

// Chunk 是文档切片
type Chunk struct {
	ID       int      `json:"id"`
	Source   string   `json:"source"`   // 来源文件
	Text     string   `json:"text"`     // 切片文本
	StartPos int      `json:"start"`    // 在原文中的起始位置
	Tokens   int      `json:"tokens"`   // 估算 token 数
	Vector   []float32 `json:"vector"`  // embedding 向量
}

// Index 是持久化的向量索引
type Index struct {
	Chunks    []Chunk `json:"chunks"`
	Dimension int     `json:"dimension"`
	Model     string  `json:"model"`
}

// ChatMessage OpenAI 兼容消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}
