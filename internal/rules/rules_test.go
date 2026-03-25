package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"printdock/internal/rules"
)

func TestLoad_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	loaded, err := rules.Load(path)
	if err != nil {
		t.Fatalf("expected no error for empty file, got: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 rules, got %d", len(loaded))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	loaded, err := rules.Load(path)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 rules for missing file, got %d", len(loaded))
	}
}

func TestLoad_WithRules(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	content := `- name: a4-rule
  match:
    filename_regex: ".*\\.pdf"
    page_width_mm: 210
    page_height_mm: 297
    dimension_tolerance_mm: 2.0
    page_count: 1
  action:
    printer: HP-LaserJet
    archive_subdir: a4
- name: letter-rule
  match:
    filename_regex: ".*\\.pdf"
    page_width_mm: 215.9
    page_height_mm: 279.4
  action:
    printer: Canon-Letter
    archive_subdir: letter
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write rules file: %v", err)
	}

	loaded, err := rules.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(loaded))
	}

	r0 := loaded[0]
	if r0.Name != "a4-rule" {
		t.Errorf("expected name=a4-rule, got %q", r0.Name)
	}
	if r0.Match.FilenameRegex != `.*\.pdf` {
		t.Errorf("unexpected FilenameRegex: %q", r0.Match.FilenameRegex)
	}
	if r0.Match.PageWidthMM != 210 {
		t.Errorf("expected PageWidthMM=210, got %v", r0.Match.PageWidthMM)
	}
	if r0.Match.PageHeightMM != 297 {
		t.Errorf("expected PageHeightMM=297, got %v", r0.Match.PageHeightMM)
	}
	if r0.Match.DimensionTolerance != 2.0 {
		t.Errorf("expected DimensionTolerance=2.0, got %v", r0.Match.DimensionTolerance)
	}
	if r0.Match.PageCount != 1 {
		t.Errorf("expected PageCount=1, got %d", r0.Match.PageCount)
	}
	if r0.Action.Printer != "HP-LaserJet" {
		t.Errorf("expected Printer=HP-LaserJet, got %q", r0.Action.Printer)
	}
	if r0.Action.ArchiveSubdir != "a4" {
		t.Errorf("expected ArchiveSubdir=a4, got %q", r0.Action.ArchiveSubdir)
	}
}

func TestStore_Load(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	content := `- name: my-rule
  match:
    page_width_mm: 210
  action:
    printer: TestPrinter
    archive_subdir: test
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write rules file: %v", err)
	}

	store := rules.NewStore(path)
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(loaded))
	}
	if loaded[0].Name != "my-rule" {
		t.Errorf("expected name=my-rule, got %q", loaded[0].Name)
	}
}

func TestStore_Add(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	store := rules.NewStore(path)

	rule := rules.Rule{
		Name: "new-rule",
		Match: rules.RuleMatch{
			PageWidthMM: 210,
		},
		Action: rules.RuleAction{
			Printer:       "SomePrinter",
			ArchiveSubdir: "subdir",
		},
	}

	if err := store.Add(rule); err != nil {
		t.Fatalf("unexpected error on Add: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error on Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 rule after Add, got %d", len(loaded))
	}
	if loaded[0].Name != "new-rule" {
		t.Errorf("expected name=new-rule, got %q", loaded[0].Name)
	}
}

func TestStore_Add_DuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	store := rules.NewStore(path)

	rule := rules.Rule{
		Name:   "dup-rule",
		Action: rules.RuleAction{Printer: "P1"},
	}

	if err := store.Add(rule); err != nil {
		t.Fatalf("unexpected error on first Add: %v", err)
	}

	err := store.Add(rule)
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	store := rules.NewStore(path)

	rule := rules.Rule{
		Name:   "to-delete",
		Action: rules.RuleAction{Printer: "P1"},
	}
	if err := store.Add(rule); err != nil {
		t.Fatalf("unexpected error on Add: %v", err)
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("unexpected error on Delete: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error on Load after Delete: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 rules after Delete, got %d", len(loaded))
	}
}

func TestStore_Reorder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, []byte(`rules:
  - name: alpha
    match: {}
    action: {printer: P1, archive_subdir: a}
  - name: beta
    match: {}
    action: {printer: P2, archive_subdir: b}
  - name: gamma
    match: {}
    action: {printer: P3, archive_subdir: c}
`), 0644)

	store := rules.NewStore(path)
	err := store.Reorder([]string{"gamma", "alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, _ := rules.Load(path)
	if loaded[0].Name != "gamma" || loaded[1].Name != "alpha" || loaded[2].Name != "beta" {
		t.Errorf("unexpected order: %v %v %v", loaded[0].Name, loaded[1].Name, loaded[2].Name)
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	store := rules.NewStore(path)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error when deleting nonexistent rule, got nil")
	}
}

func TestStore_Save_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	store := rules.NewStore(path)

	original := []rules.Rule{
		{
			Name: "rule-one",
			Match: rules.RuleMatch{
				FilenameRegex:      `.*\.pdf`,
				PageWidthMM:        210,
				PageHeightMM:       297,
				DimensionTolerance: 1.5,
				PageCount:          2,
			},
			Action: rules.RuleAction{
				Printer:       "HP",
				ArchiveSubdir: "pdfs",
			},
		},
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error on Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(loaded))
	}
	r := loaded[0]
	if r.Name != "rule-one" {
		t.Errorf("Name mismatch: got %q", r.Name)
	}
	if r.Match.FilenameRegex != `.*\.pdf` {
		t.Errorf("FilenameRegex mismatch: got %q", r.Match.FilenameRegex)
	}
	if r.Match.PageWidthMM != 210 {
		t.Errorf("PageWidthMM mismatch: got %v", r.Match.PageWidthMM)
	}
	if r.Match.DimensionTolerance != 1.5 {
		t.Errorf("DimensionTolerance mismatch: got %v", r.Match.DimensionTolerance)
	}
	if r.Match.PageCount != 2 {
		t.Errorf("PageCount mismatch: got %d", r.Match.PageCount)
	}
	if r.Action.Printer != "HP" {
		t.Errorf("Printer mismatch: got %q", r.Action.Printer)
	}
	if r.Action.ArchiveSubdir != "pdfs" {
		t.Errorf("ArchiveSubdir mismatch: got %q", r.Action.ArchiveSubdir)
	}
}
