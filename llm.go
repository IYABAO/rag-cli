package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// chatStream 调 LLM 流式输出，返回完整回复
func chatStream(baseURL, apiKey, model string, messages []ChatMessage) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"
	body, _ := json.Marshal(chatRequest{Model: model, Messages: messages, Stream: true})

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM API %d", resp.StatusCode)
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var cr chatResponse
		if json.Unmarshal([]byte(data), &cr) == nil && len(cr.Choices) > 0 {
			fmt.Print(cr.Choices[0].Delta.Content)
			full.WriteString(cr.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
	return full.String(), nil
}
