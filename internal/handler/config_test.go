package handler

import (
	"testing"

	"mathmodel-ai/internal/config"

	"gopkg.in/yaml.v3"
)

func TestUpdateOpenAIConfigWritesReasoning(t *testing.T) {
	doc := newEmptyYAMLDocument()
	allowClient := false

	updateOpenAIConfig(doc, config.OpenAIConfig{
		Provider:       "claude",
		BaseURL:        "https://api.anthropic.com",
		APIKey:         "test-key",
		Model:          "claude-3-5-sonnet",
		MaxTotalTokens: 200000,
		Reasoning: config.OpenAIReasoningConfig{
			Mode:                 "on",
			Effort:               "high",
			Profile:              "openai_compat",
			AllowClientReasoning: &allowClient,
		},
	})

	openAI := requireMapValue(t, doc.Content[0], "openai")
	if got := requireMapValue(t, openAI, "provider").Value; got != "claude" {
		t.Fatalf("provider = %q, want claude", got)
	}
	if got := requireMapValue(t, openAI, "max_total_tokens").Value; got != "200000" {
		t.Fatalf("max_total_tokens = %q, want 200000", got)
	}

	reasoning := requireMapValue(t, openAI, "reasoning")
	mode := requireMapValue(t, reasoning, "mode")
	if got := mode.Value; got != "on" {
		t.Fatalf("reasoning.mode = %q, want on", got)
	}
	if mode.Style != yaml.DoubleQuotedStyle {
		t.Fatalf("reasoning.mode should be double quoted in YAML, style = %v", mode.Style)
	}
	if got := requireMapValue(t, reasoning, "effort").Value; got != "high" {
		t.Fatalf("reasoning.effort = %q, want high", got)
	}
	if got := requireMapValue(t, reasoning, "profile").Value; got != "openai_compat" {
		t.Fatalf("reasoning.profile = %q, want openai_compat", got)
	}
	gotAllow := findBoolInMap(reasoning, "allow_client_reasoning")
	if gotAllow == nil || *gotAllow {
		t.Fatalf("reasoning.allow_client_reasoning = %v, want false", gotAllow)
	}
}

func TestUpdateOpenAIConfigPreservesReasoningExtraRequestFields(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(`
openai:
  reasoning:
    extra_request_fields:
      custom: value
`), &doc); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}

	updateOpenAIConfig(&doc, config.OpenAIConfig{
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "test-key",
		Model:    "gpt-4.1",
		Reasoning: config.OpenAIReasoningConfig{
			Mode:    "auto",
			Profile: "auto",
		},
	})

	openAI := requireMapValue(t, doc.Content[0], "openai")
	reasoning := requireMapValue(t, openAI, "reasoning")
	extra := requireMapValue(t, reasoning, "extra_request_fields")
	if got := requireMapValue(t, extra, "custom").Value; got != "value" {
		t.Fatalf("extra_request_fields.custom = %q, want value", got)
	}
}

func TestUpdateKnowledgeConfigWritesNestedSettings(t *testing.T) {
	doc := newEmptyYAMLDocument()

	updateKnowledgeConfig(doc, config.KnowledgeConfig{
		Enabled:  true,
		BasePath: "knowledge_custom",
		Embedding: config.EmbeddingConfig{
			Provider: "openai",
			BaseURL:  "https://embed.example/v1",
			APIKey:   "embed-key",
			Model:    "text-embedding-test",
		},
		Retrieval: config.RetrievalConfig{
			TopK:                8,
			SimilarityThreshold: 0.55,
			SubIndexFilter:      "prod",
			PostRetrieve: config.PostRetrieveConfig{
				PrefetchTopK:     20,
				MaxContextChars:  12000,
				MaxContextTokens: 3000,
			},
		},
		Indexing: config.IndexingConfig{
			ChunkStrategy:         "recursive",
			RequestTimeoutSeconds: 90,
			BatchSize:             32,
			PreferSourceFile:      true,
			SubIndexes:            []string{"prod", "math"},
			ChunkSize:             768,
			ChunkOverlap:          80,
			MaxChunksPerItem:      100,
			MaxRPM:                120,
			RateLimitDelayMs:      250,
			MaxRetries:            5,
			RetryDelayMs:          1500,
		},
	})

	knowledge := requireMapValue(t, doc.Content[0], "knowledge")
	if got := requireMapValue(t, knowledge, "base_path").Value; got != "knowledge_custom" {
		t.Fatalf("knowledge.base_path = %q, want knowledge_custom", got)
	}
	embedding := requireMapValue(t, knowledge, "embedding")
	if got := requireMapValue(t, embedding, "base_url").Value; got != "https://embed.example/v1" {
		t.Fatalf("knowledge.embedding.base_url = %q, want https://embed.example/v1", got)
	}
	if got := requireMapValue(t, embedding, "api_key").Value; got != "embed-key" {
		t.Fatalf("knowledge.embedding.api_key = %q, want embed-key", got)
	}

	retrieval := requireMapValue(t, knowledge, "retrieval")
	if got := requireMapValue(t, retrieval, "top_k").Value; got != "8" {
		t.Fatalf("knowledge.retrieval.top_k = %q, want 8", got)
	}
	if got := requireMapValue(t, retrieval, "similarity_threshold").Value; got != "0.55" {
		t.Fatalf("knowledge.retrieval.similarity_threshold = %q, want 0.55", got)
	}
	postRetrieve := requireMapValue(t, retrieval, "post_retrieve")
	if got := requireMapValue(t, postRetrieve, "max_context_tokens").Value; got != "3000" {
		t.Fatalf("knowledge.retrieval.post_retrieve.max_context_tokens = %q, want 3000", got)
	}

	indexing := requireMapValue(t, knowledge, "indexing")
	if got := requireMapValue(t, indexing, "chunk_strategy").Value; got != "recursive" {
		t.Fatalf("knowledge.indexing.chunk_strategy = %q, want recursive", got)
	}
	gotPrefer := findBoolInMap(indexing, "prefer_source_file")
	if gotPrefer == nil || !*gotPrefer {
		t.Fatalf("knowledge.indexing.prefer_source_file = %v, want true", gotPrefer)
	}
	subIndexes := requireMapValue(t, indexing, "sub_indexes")
	if len(subIndexes.Content) != 2 || subIndexes.Content[0].Value != "prod" || subIndexes.Content[1].Value != "math" {
		t.Fatalf("knowledge.indexing.sub_indexes = %#v, want [prod math]", subIndexes.Content)
	}
}

func requireMapValue(t *testing.T, mapNode *yaml.Node, key string) *yaml.Node {
	t.Helper()
	value := findMapValue(mapNode, key)
	if value == nil {
		t.Fatalf("missing yaml key %q", key)
	}
	return value
}
