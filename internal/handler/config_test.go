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
	if got := requireMapValue(t, reasoning, "mode").Value; got != "on" {
		t.Fatalf("reasoning.mode = %q, want on", got)
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

func requireMapValue(t *testing.T, mapNode *yaml.Node, key string) *yaml.Node {
	t.Helper()
	value := findMapValue(mapNode, key)
	if value == nil {
		t.Fatalf("missing yaml key %q", key)
	}
	return value
}
