package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// QdrantClient Qdrant REST API 客户端
type QdrantClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewQdrantClient(host string) *QdrantClient {
	return &QdrantClient{
		BaseURL:    host,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ensureCollection 确保集合存在
func (q *QdrantClient) ensureCollection(name string, dim int) error {
	url := fmt.Sprintf("%s/collections/%s", q.BaseURL, name)

	// 先检查是否已存在
	resp, err := q.HTTPClient.Get(url)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return nil // 已存在
		}
	}

	// 创建集合
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dim,
			"distance": "Cosine",
		},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err = q.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

type qdrantPoint struct {
	ID      int                    `json:"id"`
	Vector  []float32              `json:"vector,omitempty"`
	Payload map[string]interface{} `json:"payload"`
	Score   float32                `json:"score,omitempty"`
}

// UpsertPoints 批量写入向量
func (q *QdrantClient) UpsertPoints(collection string, points []qdrantPoint) error {
	url := fmt.Sprintf("%s/collections/%s/points", q.BaseURL, collection)
	body := map[string]interface{}{"points": points}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

type qdrantSearchResult struct {
	Result []struct {
		ID      int                    `json:"id"`
		Score   float32                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

// Search 向量搜索
func (q *QdrantClient) Search(collection string, vector []float32, limit int, sourceFilter string) ([]qdrantPoint, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", q.BaseURL, collection)

	reqBody := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	if sourceFilter != "" {
		reqBody["filter"] = map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "source",
					"match": map[string]interface{}{"value": sourceFilter},
				},
			},
		}
	}

	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search %d: %s", resp.StatusCode, string(rb))
	}

	var sr qdrantSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	var out []qdrantPoint
	for _, r := range sr.Result {
		out = append(out, qdrantPoint{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		})
	}
	return out, nil
}

// CountPoints 获取集合中的点数
func (q *QdrantClient) CountPoints(collection string) (int, error) {
	url := fmt.Sprintf("%s/collections/%s", q.BaseURL, collection)
	resp, err := q.HTTPClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var info struct {
		Result struct {
			PointsCount int `json:"points_count"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, err
	}
	return info.Result.PointsCount, nil
}
