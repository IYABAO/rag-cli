package main

import (
	"hash/fnv"
	"math"
)

// localEmbed 本地哈希 embedding：用字符 n-gram 生成固定维度向量
// 不依赖外部 API，能跑通完整 RAG 流程；语义精度不如真实 embedding，但可演示
func localEmbed(text string, dim int) []float32 {
	vec := make([]float32, dim)
	// 用 3-gram 滑窗
	r := []rune(text)
	for i := 0; i < len(r)-2; i++ {
		ng := string(r[i : i+3])
		h := fnv.New32a()
		h.Write([]byte(ng))
		idx := h.Sum32() % uint32(dim)
		// 符号：用 hash 的高 bit 决定正负
		sign := float32(1)
		if (h.Sum32()>>16)%2 == 1 {
			sign = -1
		}
		vec[idx] += sign
	}
	// 归一化
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}
