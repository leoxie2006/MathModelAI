package database

import (
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestProjectFilesCRUD(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "test.db"), zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	p, err := db.CreateProject(&Project{Name: "workspace test"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := db.UpsertProjectFile(&ProjectFile{
		ProjectID:    p.ID,
		OriginalName: "data.csv",
		StoredName:   "data.csv",
		RelPath:      "data/data.csv",
		AbsPath:      "/tmp/data.csv",
		FileType:     "data",
		MimeType:     "text/csv",
		SizeBytes:    12,
		SHA256:       "abc",
		Source:       "upload",
	})
	if err != nil {
		t.Fatal(err)
	}
	if file.ID == "" {
		t.Fatal("expected id")
	}
	list, err := db.ListProjectFiles(p.ID, "data", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].RelPath != "data/data.csv" {
		t.Fatalf("unexpected list: %#v", list)
	}
	updated, err := db.UpsertProjectFile(&ProjectFile{
		ProjectID:    p.ID,
		OriginalName: "data.csv",
		StoredName:   "data.csv",
		RelPath:      "data/data.csv",
		AbsPath:      "/tmp/data.csv",
		FileType:     "data",
		MimeType:     "text/csv",
		SizeBytes:    20,
		SHA256:       "def",
		Source:       "scan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != file.ID || updated.SizeBytes != 20 || updated.SHA256 != "def" {
		t.Fatalf("upsert did not update existing row: %#v", updated)
	}
	got, err := db.GetProjectFileByRelPath(p.ID, "data/data.csv")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != file.ID {
		t.Fatalf("unexpected file id: %s", got.ID)
	}
	if err := db.DeleteProjectFile(p.ID, file.ID); err != nil {
		t.Fatal(err)
	}
	list, err = db.ListProjectFiles(p.ID, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %#v", list)
	}
}
