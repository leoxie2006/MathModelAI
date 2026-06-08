package database

import (
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestUpsertProjectFact_preservesBodyOnEmptyUpdate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "facts.db")
	db, err := NewDB(dbPath, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	proj, err := db.CreateProject(&Project{Name: "test-facts"})
	if err != nil {
		t.Fatal(err)
	}

	const body = "## 模型规格\n- 目标: 多指标评价\n## 验证方案\n- 基线: 等权 TOPSIS\n"
	_, err = db.UpsertProjectFact(&ProjectFact{
		ProjectID: proj.ID,
		FactKey:   "model/topsis-plan",
		Category:  "model",
		Summary:   "P1 使用 TOPSIS 做多指标评价",
		Body:      body,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := db.UpsertProjectFact(&ProjectFact{
		ProjectID: proj.ID,
		FactKey:   "model/topsis-plan",
		Summary:   "P1 使用 TOPSIS 做多指标评价（已确认）",
		Body:      "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Summary != "P1 使用 TOPSIS 做多指标评价（已确认）" {
		t.Fatalf("summary=%q", updated.Summary)
	}
	if updated.Body != body {
		t.Fatalf("returned body=%q want preserved modeling card", updated.Body)
	}

	fromDB, err := db.GetProjectFactByKey(proj.ID, "model/topsis-plan")
	if err != nil {
		t.Fatal(err)
	}
	if fromDB.Body != body {
		t.Fatalf("stored body=%q want preserved", fromDB.Body)
	}
}

func TestUpsertProjectFact_replacesBodyWhenProvided(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "facts.db")
	db, err := NewDB(dbPath, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	proj, err := db.CreateProject(&Project{Name: "test-facts"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.UpsertProjectFact(&ProjectFact{
		ProjectID: proj.ID,
		FactKey:   "problem/requirements",
		Summary:   "v1",
		Body:      "old body",
	})
	if err != nil {
		t.Fatal(err)
	}

	const newBody = "new body with evidence"
	updated, err := db.UpsertProjectFact(&ProjectFact{
		ProjectID: proj.ID,
		FactKey:   "problem/requirements",
		Summary:   "v2",
		Body:      newBody,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Body != newBody {
		t.Fatalf("body=%q want %q", updated.Body, newBody)
	}
}

func TestRestoreProjectFact(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "facts.db")
	db, err := NewDB(dbPath, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	proj, err := db.CreateProject(&Project{Name: "restore-test"})
	if err != nil {
		t.Fatal(err)
	}
	key := "problem/restore-me"
	_, err = db.UpsertProjectFact(&ProjectFact{
		ProjectID:  proj.ID,
		FactKey:    key,
		Summary:    "s",
		Confidence: "confirmed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DeprecateProjectFact(proj.ID, key); err != nil {
		t.Fatal(err)
	}
	if err := db.RestoreProjectFact(proj.ID, key, "confirmed"); err != nil {
		t.Fatal(err)
	}
	f, err := db.GetProjectFactByKey(proj.ID, key)
	if err != nil {
		t.Fatal(err)
	}
	if f.Confidence != "confirmed" {
		t.Fatalf("confidence=%q want confirmed", f.Confidence)
	}
	if err := db.RestoreProjectFact(proj.ID, key, ""); err == nil {
		t.Fatal("expected error when not deprecated")
	}
}

func TestUpsertProjectFact_createsVersionOnContentChange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "facts.db")
	db, err := NewDB(dbPath, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	proj, err := db.CreateProject(&Project{Name: "version-test"})
	if err != nil {
		t.Fatal(err)
	}

	created, err := db.UpsertProjectFact(&ProjectFact{
		ProjectID: proj.ID,
		FactKey:   "model/version-test",
		Category:  "model",
		Summary:   "v1",
		Body:      "body v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.SupersedesFactID != "" {
		t.Fatalf("expected no supersedes on create, got %q", created.SupersedesFactID)
	}

	updated, err := db.UpsertProjectFact(&ProjectFact{
		ProjectID: proj.ID,
		FactKey:   "model/version-test",
		Summary:   "v2",
		Body:      "body v2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.SupersedesFactID == "" {
		t.Fatal("expected supersedes_fact_id after content change")
	}
	prev, err := db.GetProjectFactVersion(updated.SupersedesFactID)
	if err != nil {
		t.Fatal(err)
	}
	if prev.Summary != "v1" || prev.Body != "body v1" {
		t.Fatalf("previous version mismatch: summary=%q body=%q", prev.Summary, prev.Body)
	}
}

func TestMergeFactBodyOnUpdate(t *testing.T) {
	if got := mergeFactBodyOnUpdate("", "keep"); got != "keep" {
		t.Fatalf("empty incoming: got %q", got)
	}
	if got := mergeFactBodyOnUpdate("  ", "keep"); got != "keep" {
		t.Fatalf("whitespace incoming: got %q", got)
	}
	if got := mergeFactBodyOnUpdate("new", "old"); got != "new" {
		t.Fatalf("non-empty incoming: got %q", got)
	}
}
