# rag-cli — 用 Go 标准库写一个 RAG 知识库问答 CLI

纯 Go 标准库实现的检索增强生成（RAG）命令行工具，零第三方依赖。
支持 Qdrant 向量数据库 + DeepSeek/OpenAI 兼容 API。

## 架构

```
文档(.md/.txt/.pdf) → 切片(段落+句子边界, 800字符/重叠80)
  → Embedding(本地哈希 或 API)
  → 向量存储(Qdrant / JSON 文件)
  → 重复导入自动跳过(文件修改时间缓存)
  → Cosine 相似度 Top-K 检索
  → 拼 Prompt → LLM SSE 流式回答
```

## 快速开始

### 1. 启动 Qdrant

```bash
docker compose up -d
# 或
docker run -d -p 6333:6333 qdrant/qdrant
```

### 2. 配置 API Key

```powershell
# Windows
$env:RAG_API_KEY="sk-你的deepseek key"

# Linux/Mac
export RAG_API_KEY="sk-xxx"
```

### 3. 导入知识库

```bash
# 导入单个文件（支持 .md/.txt/.pdf）
./rag-cli --local --qdrant ingest notes.md

# 导入整个目录（自动遍历）
./rag-cli --local --qdrant ingest ./docs/

# 重复导入自动跳过未变化的文件
./rag-cli --local --qdrant ingest ./docs/
```

### 4. 提问

```bash
# 基础问答
./rag-cli --local --qdrant ask "context 怎么取消 goroutine？"

# 按文件名过滤检索
./rag-cli --local --qdrant ask "超时怎么设？" --source notes.md
```

## 命令行参数

| 参数 | 说明 |
|------|------|
| `--local` | 使用本地哈希 embedding（256维，不需要 Embedding API） |
| `--qdrant` | 使用 Qdrant 向量数据库（默认 JSON 文件存储） |
| `ingest <路径>` | 导入文件或目录 |
| `ask "<问题>"` | 基于知识库提问 |
| `--source <文件名>` | 按文件名过滤检索范围 |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `RAG_BASE_URL` | `https://api.deepseek.com/v1` | LLM API 地址 |
| `RAG_API_KEY` | （空） | API Key |
| `RAG_MODEL` | `deepseek-chat` | 聊天模型 |
| `RAG_EMBED_MODEL` | `text-embedding-3-small` | Embedding 模型 |
| `QDRANT_URL` | `http://localhost:6333` | Qdrant 地址 |
| `QDRANT_COLLECTION` | `rag_knowledge` | Qdrant 集合名 |

## 技术要点

- **零第三方依赖**：纯标准库 net/http，编译后单文件 8MB
- **双后端存储**：Qdrant（生产）/ JSON 文件（轻量）
- **多格式支持**：Markdown、TXT、PDF（`ledongthuc/pdf`）
- **智能切片**：按段落和句子边界切分，重叠 80 字符保留上下文
- **重复导入缓存**：记录文件修改时间，未变化的自动跳过
- **SSE 流式输出**：逐 token 打印 LLM 回答
- **批量导入**：目录遍历，自动识别 .md/.txt/.markdown
- **元数据过滤**：按 source 文件名过滤检索

## 项目结构

```
rag-cli/
├── main.go          # CLI 入口，参数解析，ingest/ask 流程
├── chunker.go       # 文档切片（段落+句子边界）
├── embedding.go     # Embedding API 调用 + 余弦相似度
├── local_embed.go   # 本地哈希 embedding（FNV + 3-gram）
├── llm.go           # LLM SSE 流式调用
├── qdrant.go        # Qdrant REST API 客户端
├── pdf.go           # PDF 文本提取（ledongthuc/pdf）
├── cache.go         # 导入缓存（文件修改时间跳过）
├── types.go         # 数据结构定义
├── docker-compose.yml
├── Makefile
└── notes.md         # 示例文档
```

## License

MIT
