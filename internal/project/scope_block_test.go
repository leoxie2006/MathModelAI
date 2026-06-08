package project

import (
	"strings"
	"testing"

	"mathmodel-ai/internal/database"
)

func TestBuildScopeBlock_targetsExcludeNotes(t *testing.T) {
	proj := &database.Project{
		ID:        "p1",
		Name:      "MCM-2026-A",
		ScopeJSON: `{"targets":["问题一"],"exclude":["外部付费数据"],"data_files":["附件1.xlsx"],"deliverables":["paper.pdf"],"constraints":["只能使用题目附件"],"deadline":"2026-06-10 20:00","notes":"先完成可解释模型"}`,
	}
	block := BuildScopeBlock(proj)
	if !strings.Contains(block, "问题一") {
		t.Fatalf("missing target: %s", block)
	}
	if !strings.Contains(block, "外部付费数据") {
		t.Fatalf("missing exclude: %s", block)
	}
	for _, want := range []string{"附件1.xlsx", "paper.pdf", "只能使用题目附件", "2026-06-10 20:00", "先完成可解释模型"} {
		if !strings.Contains(block, want) {
			t.Fatalf("missing %q: %s", want, block)
		}
	}
	if strings.Contains(block, "扫描") || strings.Contains(block, "利用") {
		t.Fatalf("scope block still contains security wording: %s", block)
	}
}

func TestBuildScopeBlock_legacyTargetsRemainCompatible(t *testing.T) {
	proj := &database.Project{
		ID:        "p1",
		Name:      "legacy",
		ScopeJSON: `{"targets":["附件 A"],"exclude":["未授权数据"],"notes":"仅题面数据"}`,
	}
	block := BuildScopeBlock(proj)
	if !strings.Contains(block, "附件 A") {
		t.Fatalf("missing target: %s", block)
	}
	if !strings.Contains(block, "未授权数据") {
		t.Fatalf("missing exclude: %s", block)
	}
	if !strings.Contains(block, "仅题面数据") {
		t.Fatalf("missing notes: %s", block)
	}
}

func TestBuildScopeBlock_empty(t *testing.T) {
	if BuildScopeBlock(&database.Project{Name: "X"}) != "" {
		t.Fatal("expected empty")
	}
}

func TestBuildScopeBlock_invalidJSON(t *testing.T) {
	proj := &database.Project{Name: "X", ScopeJSON: `{not json`}
	block := BuildScopeBlock(proj)
	if !strings.Contains(block, "非合法 JSON") {
		t.Fatalf("unexpected: %s", block)
	}
}
