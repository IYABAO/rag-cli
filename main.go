package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := loadConfig()

	subCmd := ""
	var subArgs []string
	for _, a := range os.Args[1:] {
		if a == "--local" || a == "--qdrant" || a == "--help" || a == "-h" {
			continue
		}
		if subCmd == "" {
			subCmd = a
		} else {
			subArgs = append(subArgs, a)
		}
	}

	switch subCmd {
	case "ingest":
		if len(subArgs) < 1 {
			fmt.Println("用法: rag ingest <文件或目录>")
			os.Exit(1)
		}
		runIngest(subArgs[0], cfg)
	case "ask":
		if len(subArgs) < 1 {
			fmt.Println("用法: rag ask \"你的问题\" [--source 文件名]")
			os.Exit(1)
		}
		// 解析 --source 过滤
		sourceFilter := ""
		var questionParts []string
		for _, a := range subArgs {
			if a == "--source" {
				continue
			}
			if len(questionParts) > 0 && questionParts[len(questionParts)-1] == "--source" {
				sourceFilter = a
				continue
			}
			questionParts = append(questionParts, a)
		}
		runAsk(strings.Join(questionParts, " "), sourceFilter, cfg)
	default:
		printUsage()
	}
}

type config struct {
	BaseURL       string
	APIKey        string
	ChatModel     string
	EmbedModel    string
	IndexFile     string
	MaxChunkChars int
	Overlap       int
	TopK          int
	LocalEmbed    bool
	UseQdrant     bool
	QdrantURL     string
	Collection    string
}

func loadConfig() *config {
	c := &config{
		BaseURL:       getenv("RAG_BASE_URL", "https://api.deepseek.com/v1"),
		APIKey:        getenv("RAG_API_KEY", ""),
		ChatModel:     getenv("RAG_MODEL", "deepseek-chat"),
		EmbedModel:    getenv("RAG_EMBED_MODEL", "text-embedding-3-small"),
		IndexFile:     "rag_index.json",
		MaxChunkChars: 800,
		Overlap:       80,
		TopK:          3,
		QdrantURL:     getenv("QDRANT_URL", "http://localhost:6333"),
		Collection:    getenv("QDRANT_COLLECTION", "rag_knowledge"),
	}
	for _, a := range os.Args {
		if a == "--local" {
			c.LocalEmbed = true
		}
		if a == "--qdrant" {
			c.UseQdrant = true
		}
	}
	if c.APIKey == "" && !c.LocalEmbed {
		fmt.Fprintln(os.Stderr, "警告: RAG_API_KEY 未设置，或用 --local 启用本地 embedding")
	}
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func printUsage() {
	fmt.Println(`rag-cli — 本地知识库问答

环境变量:
  RAG_BASE_URL      API 地址 (默认 https://api.deepseek.com/v1)
  RAG_API_KEY       API Key
  RAG_MODEL         聊天模型 (默认 deepseek-chat)
  RAG_EMBED_MODEL   Embedding 模型 (默认 text-embedding-3-small)
  QDRANT_URL        Qdrant 地址 (默认 http://localhost:6333)
  QDRANT_COLLECTION 集合名 (默认 rag_knowledge)

用法:
  rag [--qdrant] ingest <文件或目录>   导入知识库（目录批量导入 .md/.txt）
  rag [--qdrant] ask "<问题>"          基于知识库提问
  rag [--qdrant] ask "<问题>" --source notes.md   按文件名过滤`)
}

// collectFiles 收集目录下所有 .md/.txt 文件
func collectFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	var files []string
	filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".md" || ext == ".txt" || ext == ".markdown" || ext == ".pdf" {
			files = append(files, p)
		}
		return nil
	})
	return files, nil
}

func runIngest(path string, cfg *config) {
	files, err := collectFiles(path)
	if err != nil {
		fmt.Println("路径错误:", err)
		os.Exit(1)
	}
	fmt.Printf("发现 %d 个文件\n", len(files))

	// 加载导入日志，缓存未变化的文件
	log := loadImportLog("import_log.json")
	fmt.Println(log.summary())

	// 收集所有切片
	var allChunks []Chunk
	for _, f := range files {
		if !log.isFileChanged(f) {
			fmt.Printf("  跳过(未变化): %s\n", f)
			continue
		}
		text, err := loadFile(f)
		if err != nil {
			fmt.Printf("  跳过 %s: %v\n", f, err)
			continue
		}
		chunks := chunkText(text, filepath.Base(f), cfg.MaxChunkChars, cfg.Overlap)
		fmt.Printf("  %s: %d 段\n", f, len(chunks))
		log.markImported(f, len(chunks))
		allChunks = append(allChunks, chunks...)
	}
	saveImportLog("import_log.json", log)
	fmt.Printf("共 %d 段需要 embedding...\n", len(allChunks))

	if len(allChunks) == 0 {
		fmt.Println("没有新内容需要导入")
		return
	}

	// Embedding
	for i := range allChunks {
		fmt.Printf("  %d/%d\r", i+1, len(allChunks))
		if cfg.LocalEmbed {
			allChunks[i].Vector = localEmbed(allChunks[i].Text, 256)
		} else {
			vec, err := embed(allChunks[i].Text, cfg.BaseURL, cfg.APIKey, cfg.EmbedModel)
			if err != nil {
				fmt.Printf("\nembedding 失败 [%s]: %v\n", allChunks[i].Source, err)
				os.Exit(1)
			}
			allChunks[i].Vector = vec
		}
	}
	fmt.Println()

	if cfg.UseQdrant {
		ingestQdrant(allChunks, cfg)
	} else {
		ingestJSON(allChunks, cfg)
	}
}

func ingestQdrant(chunks []Chunk, cfg *config) {
	qc := NewQdrantClient(cfg.QdrantURL)
	dim := 256
	if len(chunks) > 0 && len(chunks[0].Vector) > 0 {
		dim = len(chunks[0].Vector)
	}

	if err := qc.ensureCollection(cfg.Collection, dim); err != nil {
		fmt.Println("Qdrant 连接失败:", err)
		fmt.Println("请先启动: docker run -p 6333:6333 qdrant/qdrant")
		os.Exit(1)
	}

	// 获取当前最大 ID（追加模式）
	existing, _ := qc.CountPoints(cfg.Collection)
	fmt.Printf("集合 %s 现有 %d 段\n", cfg.Collection, existing)

	points := make([]qdrantPoint, len(chunks))
	for i, c := range chunks {
		points[i] = qdrantPoint{
			ID:     existing + i,
			Vector: c.Vector,
			Payload: map[string]interface{}{
				"source":    c.Source,
				"text":      c.Text,
				"chunk_id":  c.ID,
				"tokens":    c.Tokens,
			},
		}
	}

	if err := qc.UpsertPoints(cfg.Collection, points); err != nil {
		fmt.Println("写入 Qdrant 失败:", err)
		os.Exit(1)
	}

	total, _ := qc.CountPoints(cfg.Collection)
	fmt.Printf("完成: %d 段已写入 Qdrant，集合共 %d 段\n", len(chunks), total)
}

func ingestJSON(chunks []Chunk, cfg *config) {
	idx := loadIndex(cfg.IndexFile)
	startID := len(idx.Chunks)
	for i := range chunks {
		chunks[i].ID = startID + i
	}
	idx.Chunks = append(idx.Chunks, chunks...)
	if len(chunks) > 0 {
		idx.Dimension = len(chunks[0].Vector)
	}
	idx.Model = cfg.EmbedModel
	saveIndex(cfg.IndexFile, idx)
	fmt.Printf("完成: 共 %d 段向量已存入 %s\n", len(idx.Chunks), cfg.IndexFile)
}

func runAsk(question, sourceFilter string, cfg *config) {
	// 问题 embedding
	var qVec []float32
	if cfg.LocalEmbed {
		qVec = localEmbed(question, 256)
	} else {
		var err error
		qVec, err = embed(question, cfg.BaseURL, cfg.APIKey, cfg.EmbedModel)
		if err != nil {
			fmt.Println("问题 embedding 失败:", err)
			os.Exit(1)
		}
	}

	var context string

	if cfg.UseQdrant {
		qc := NewQdrantClient(cfg.QdrantURL)
		results, err := qc.Search(cfg.Collection, qVec, cfg.TopK, sourceFilter)
		if err != nil {
			fmt.Println("Qdrant 搜索失败:", err)
			os.Exit(1)
		}
		if len(results) == 0 {
			fmt.Println("未检索到相关片段")
			os.Exit(0)
		}
		fmt.Printf("=== 检索到 Top-%d 片段 (Qdrant) ===\n", len(results))
		for i, r := range results {
			source, _ := r.Payload["source"].(string)
			text, _ := r.Payload["text"].(string)
			fmt.Printf("[score=%.3f] %s\n", r.Score, source)
			context += fmt.Sprintf("--- 片段%d (来自 %s) ---\n%s\n\n", i+1, source, text)
		}
	} else {
		idx := loadIndex(cfg.IndexFile)
		if len(idx.Chunks) == 0 {
			fmt.Println("知识库为空，先运行 rag ingest <文件>")
			os.Exit(1)
		}
		type scored struct {
			chunk Chunk
			score float32
		}
		var scores []scored
		for _, c := range idx.Chunks {
			if sourceFilter != "" && c.Source != sourceFilter {
				continue
			}
			scores = append(scores, scored{chunk: c, score: cosineSimilarity(qVec, c.Vector)})
		}
		sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
		k := cfg.TopK
		if k > len(scores) {
			k = len(scores)
		}
		if k == 0 {
			fmt.Println("未检索到相关片段")
			os.Exit(0)
		}
		fmt.Printf("=== 检索到 Top-%d 片段 (本地JSON) ===\n", k)
		for i := 0; i < k; i++ {
			fmt.Printf("[score=%.3f] %s\n", scores[i].score, scores[i].chunk.Source)
			context += fmt.Sprintf("--- 片段%d (来自 %s) ---\n%s\n\n", i+1, scores[i].chunk.Source, scores[i].chunk.Text)
		}
	}

	fmt.Println("=== AI 回答 ===")
	prompt := fmt.Sprintf(`你是一个知识库问答助手。根据以下参考资料回答问题。
如果资料里没有答案，直接说"根据现有资料无法回答"，不要编造。

参考资料：
%s

问题：%s`, context, question)

	msgs := []ChatMessage{{Role: "user", Content: prompt}}
	chatStream(cfg.BaseURL, cfg.APIKey, cfg.ChatModel, msgs)
}

func loadIndex(path string) *Index {
	b, err := os.ReadFile(path)
	if err != nil {
		return &Index{}
	}
	var idx Index
	json.Unmarshal(b, &idx)
	return &idx
}

func saveIndex(path string, idx *Index) {
	b, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(path, b, 0644)
}
