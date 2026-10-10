package main

import (
	"strings"
)

// chunkText 按段落切片，超长段落按句子二次切分
func chunkText(text string, source string, maxChars int, overlap int) []Chunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// 先按双换行分段落
	paragraphs := strings.Split(text, "\n\n")
	var chunks []Chunk
	id := 0

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 短段落直接作为一片
		if len(p) <= maxChars {
			chunks = append(chunks, Chunk{ID: id, Source: source, Text: p, Tokens: estimateTokens(p)})
			id++
			continue
		}
		// 长段落按句子切分
		sentences := splitBySentence(p)
		var cur string
		for _, s := range sentences {
			if len(cur)+len(s) > maxChars && cur != "" {
				chunks = append(chunks, Chunk{ID: id, Source: source, Text: strings.TrimSpace(cur), Tokens: estimateTokens(cur)})
				id++
				// 重叠：保留最后 overlap 个字符
				r := []rune(cur)
				if len(r) > overlap {
					cur = string(r[len(r)-overlap:]) + s
				} else {
					cur = s
				}
			} else {
				cur += s
			}
		}
		if strings.TrimSpace(cur) != "" {
			chunks = append(chunks, Chunk{ID: id, Source: source, Text: strings.TrimSpace(cur), Tokens: estimateTokens(cur)})
			id++
		}
	}
	return chunks
}

// splitBySentence 按中英文句子边界切分
func splitBySentence(text string) []string {
	var parts []string
	var buf strings.Builder
	for _, r := range text {
		buf.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' || r == '\n' {
			s := buf.String()
			if strings.TrimSpace(s) != "" {
				parts = append(parts, s)
			}
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts
}

// estimateTokens 粗略估算：中文按字，英文按词
func estimateTokens(s string) int {
	r := []rune(s)
	// 粗略：每 1.5 个字符约 1 token
	return (len(r) + 1) / 2
}
