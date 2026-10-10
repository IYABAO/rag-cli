package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// readPDF 提取 PDF 文本内容
func readPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开PDF失败: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	totalPages := r.NumPage()
	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content := page.Content()
		for _, text := range content.Text {
			buf.WriteString(text.S)
			buf.WriteString(" ")
		}
		buf.WriteString("\n\n")
	}

	text := strings.TrimSpace(buf.String())
	if text == "" {
		return "", fmt.Errorf("PDF未提取到文本（可能是扫描件）")
	}
	return text, nil
}

// loadFile 统一文件读取，自动识别扩展名
func loadFile(path string) (string, error) {
	ext := strings.ToLower(strings.TrimSuffix(filepath.Ext(path), ""))
	switch ext {
	case ".pdf":
		return readPDF(path)
	default:
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
