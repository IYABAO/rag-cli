package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// importRecord 记录已导入文件的状态
type importRecord struct {
	Path    string    `json:"path"`
	ModTime time.Time `json:"mod_time"`
	Chunks  int       `json:"chunks"`
}

// importLog 导入日志
type importLog struct {
	Files map[string]importRecord `json:"files"`
}

func loadImportLog(path string) *importLog {
	b, err := os.ReadFile(path)
	if err != nil {
		return &importLog{Files: map[string]importRecord{}}
	}
	var log importLog
	json.Unmarshal(b, &log)
	if log.Files == nil {
		log.Files = map[string]importRecord{}
	}
	return &log
}

func saveImportLog(path string, log *importLog) {
	b, _ := json.MarshalIndent(log, "", "  ")
	os.WriteFile(path, b, 0644)
}

// isFileChanged 检查文件是否需要重新导入
func (l *importLog) isFileChanged(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	rec, ok := l.Files[filepath.Clean(path)]
	if !ok {
		return true
	}
	return info.ModTime().After(rec.ModTime)
}

// markImported 记录文件已导入
func (l *importLog) markImported(path string, chunks int) {
	info, err := os.Stat(path)
	modTime := time.Now()
	if err == nil {
		modTime = info.ModTime()
	}
	l.Files[filepath.Clean(path)] = importRecord{
		Path:    filepath.Clean(path),
		ModTime: modTime,
		Chunks:  chunks,
	}
}

func (l *importLog) summary() string {
	return fmt.Sprintf("已缓存 %d 个文件", len(l.Files))
}
