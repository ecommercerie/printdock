package processor

import (
	"path/filepath"
	"testing"
	"time"

	"printdock/internal/archiver"
	"printdock/internal/classifier"
	"printdock/internal/history"
	"printdock/internal/printer"
	"printdock/internal/rules"
)

// mockPrinter is a test double for the printer.Printer interface.
type mockPrinter struct {
	printed []string
}

func (m *mockPrinter) Print(path, printerName string) error {
	m.printed = append(m.printed, path)
	return nil
}

func (m *mockPrinter) ListPrinters() ([]string, error)  { return nil, nil }
func (m *mockPrinter) IsOnline(printerName string) bool  { return true }
func (m *mockPrinter) TestPrint(printerName string) error { return nil }
func (m *mockPrinter) IsSumatraInstalled() bool           { return true }
func (m *mockPrinter) DownloadSumatra() error             { return nil }
func (m *mockPrinter) GetSumatraStatus() printer.SumatraStatus {
	return printer.SumatraStatus{Installed: true, CurrentVersion: "3.5.2", LatestVersion: "3.5.2"}
}
func (m *mockPrinter) Close() error                       { return nil }

// setupTest creates all dependencies backed by a temp directory and returns
// a ready-to-use Processor along with its channels and the history DB.
func setupTest(t *testing.T, rulesList []rules.Rule) (
	*Processor,
	chan string,
	chan LearningItem,
	*mockPrinter,
	*history.DB,
) {
	t.Helper()
	dir := t.TempDir()

	fileChan := make(chan string, 8)
	learningChan := make(chan LearningItem, 8)

	cl := classifier.New(rulesList)
	mp := &mockPrinter{}
	ar := archiver.New(filepath.Join(dir, "archive"))

	h, err := history.Open(filepath.Join(dir, "history.db"))
	if err != nil {
		t.Fatalf("history.Open: %v", err)
	}
	t.Cleanup(func() { h.Close() })

	proc := New(fileChan, learningChan, cl, mp, ar, h, 1)
	return proc, fileChan, learningChan, mp, h
}

// TestProcessFile_MatchingRule verifies that a file matching a rule is printed,
// archived with status "printed", and recorded in history with status "printed".
func TestProcessFile_MatchingRule(t *testing.T) {
	rulesList := []rules.Rule{
		{
			Name: "a4-rule",
			Match: rules.RuleMatch{
				PageWidthMM:        210,
				PageHeightMM:       297,
				DimensionTolerance: 10,
			},
			Action: rules.RuleAction{
				Printer:       "TestPrinter",
				ArchiveSubdir: "a4",
			},
		},
	}

	proc, fileChan, _, mp, h := setupTest(t, rulesList)
	proc.Start()

	// Create a valid A4 PDF (595.28 x 841.89 points ≈ 210 x 297 mm).
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "document.pdf")
	if err := createTestPDF(pdfPath, 595.28, 841.89); err != nil {
		t.Fatalf("createTestPDF: %v", err)
	}

	fileChan <- pdfPath

	// Give the worker time to process the file.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := h.Query(history.HistoryFilter{Limit: 10})
		if err != nil {
			t.Fatalf("history.Query: %v", err)
		}
		if len(entries) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	proc.Stop()

	// Verify printer was called.
	if len(mp.printed) == 0 {
		t.Fatal("expected printer.Print to be called, but it was not")
	}

	// Verify history entry.
	entries, err := h.Query(history.HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("history.Query: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected a history entry, got none")
	}
	e := entries[0]
	if e.Status != "printed" {
		t.Errorf("expected status %q, got %q", "printed", e.Status)
	}
	if e.Printer != "TestPrinter" {
		t.Errorf("expected printer %q, got %q", "TestPrinter", e.Printer)
	}
	if e.RuleName != "a4-rule" {
		t.Errorf("expected rule name %q, got %q", "a4-rule", e.RuleName)
	}
	if e.ArchivePath == "" {
		t.Error("expected a non-empty archive path")
	}
}

// TestProcessFile_NoMatchingRule verifies that a file with no matching rule is
// pushed onto the learning channel and does NOT get printed or inserted into history.
func TestProcessFile_NoMatchingRule(t *testing.T) {
	// Empty rule set → nothing will match.
	proc, fileChan, learningChan, mp, h := setupTest(t, []rules.Rule{})
	proc.Start()

	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "unknown.pdf")
	// A5 dimensions (148 x 210 mm ≈ 419.53 x 595.28 points).
	if err := createTestPDF(pdfPath, 419.53, 595.28); err != nil {
		t.Fatalf("createTestPDF: %v", err)
	}

	fileChan <- pdfPath

	// Wait for the learning item to appear.
	var item LearningItem
	select {
	case item = <-learningChan:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for learning item")
	}

	proc.Stop()

	// Learning item sanity checks.
	if item.Filename != "unknown.pdf" {
		t.Errorf("expected filename %q, got %q", "unknown.pdf", item.Filename)
	}
	if item.ID == "" {
		t.Error("expected a non-empty learning item ID")
	}
	if item.PageCount != 1 {
		t.Errorf("expected PageCount 1, got %d", item.PageCount)
	}

	// Printer must NOT have been called.
	if len(mp.printed) != 0 {
		t.Errorf("expected printer not to be called, but got %v", mp.printed)
	}

	// History must remain empty.
	entries, err := h.Query(history.HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("history.Query: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no history entries, got %d", len(entries))
	}
}
