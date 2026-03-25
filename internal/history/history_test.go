package history_test

import (
	"path/filepath"
	"testing"
	"time"

	"printdock/internal/history"
)

func openTestDB(t *testing.T) *history.DB {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := history.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertAndQuery(t *testing.T) {
	db := openTestDB(t)

	e := history.HistoryEntry{
		Filename:     "test.pdf",
		OriginalPath: "/watch/test.pdf",
		ArchivePath:  "/archive/test.pdf",
		Printer:      "HP-LaserJet",
		RuleName:     "default",
		Status:       "printed",
		PageCount:    3,
		PageWidthMM:  210.0,
		PageHeightMM: 297.0,
		ErrorMessage: "",
	}

	if err := db.Insert(e); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	entries, err := db.Query(history.HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Filename != e.Filename {
		t.Errorf("Filename: got %q, want %q", got.Filename, e.Filename)
	}
	if got.OriginalPath != e.OriginalPath {
		t.Errorf("OriginalPath: got %q, want %q", got.OriginalPath, e.OriginalPath)
	}
	if got.ArchivePath != e.ArchivePath {
		t.Errorf("ArchivePath: got %q, want %q", got.ArchivePath, e.ArchivePath)
	}
	if got.Printer != e.Printer {
		t.Errorf("Printer: got %q, want %q", got.Printer, e.Printer)
	}
	if got.RuleName != e.RuleName {
		t.Errorf("RuleName: got %q, want %q", got.RuleName, e.RuleName)
	}
	if got.Status != e.Status {
		t.Errorf("Status: got %q, want %q", got.Status, e.Status)
	}
	if got.PageCount != e.PageCount {
		t.Errorf("PageCount: got %d, want %d", got.PageCount, e.PageCount)
	}
	if got.PageWidthMM != e.PageWidthMM {
		t.Errorf("PageWidthMM: got %f, want %f", got.PageWidthMM, e.PageWidthMM)
	}
	if got.PageHeightMM != e.PageHeightMM {
		t.Errorf("PageHeightMM: got %f, want %f", got.PageHeightMM, e.PageHeightMM)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID after insert")
	}
}

func TestFilterByStatus(t *testing.T) {
	db := openTestDB(t)

	entries := []history.HistoryEntry{
		{Filename: "a.pdf", OriginalPath: "/a.pdf", Status: "printed"},
		{Filename: "b.pdf", OriginalPath: "/b.pdf", Status: "failed"},
		{Filename: "c.pdf", OriginalPath: "/c.pdf", Status: "printed"},
		{Filename: "d.pdf", OriginalPath: "/d.pdf", Status: "pending"},
	}
	for _, e := range entries {
		if err := db.Insert(e); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	printed, err := db.Query(history.HistoryFilter{Status: "printed", Limit: 50})
	if err != nil {
		t.Fatalf("Query printed: %v", err)
	}
	if len(printed) != 2 {
		t.Errorf("expected 2 printed entries, got %d", len(printed))
	}

	failed, err := db.Query(history.HistoryFilter{Status: "failed", Limit: 50})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("expected 1 failed entry, got %d", len(failed))
	}

	all, err := db.Query(history.HistoryFilter{Limit: 50})
	if err != nil {
		t.Fatalf("Query all: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 total entries, got %d", len(all))
	}
}

func TestDefaultLimit(t *testing.T) {
	db := openTestDB(t)

	// Insert 60 entries
	for i := 0; i < 60; i++ {
		e := history.HistoryEntry{
			Filename:     "file.pdf",
			OriginalPath: "/file.pdf",
			Status:       "printed",
		}
		if err := db.Insert(e); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	// Zero limit should default to 50
	results, err := db.Query(history.HistoryFilter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 50 {
		t.Errorf("expected default limit of 50, got %d", len(results))
	}
}

func TestQueryOrderByCreatedAtDesc(t *testing.T) {
	db := openTestDB(t)

	for _, name := range []string{"first.pdf", "second.pdf", "third.pdf"} {
		if err := db.Insert(history.HistoryEntry{
			Filename:     name,
			OriginalPath: "/" + name,
			Status:       "printed",
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
		// Small sleep to ensure distinct timestamps
		time.Sleep(10 * time.Millisecond)
	}

	results, err := db.Query(history.HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// First result should be the most recently inserted
	if results[0].Filename != "third.pdf" {
		t.Errorf("expected most recent first, got %q", results[0].Filename)
	}
	if results[2].Filename != "first.pdf" {
		t.Errorf("expected oldest last, got %q", results[2].Filename)
	}
}

func TestGetByID(t *testing.T) {
	db := openTestDB(t)

	e := history.HistoryEntry{
		Filename:     "lookup.pdf",
		OriginalPath: "/lookup.pdf",
		Status:       "printed",
		PageCount:    7,
	}
	if err := db.Insert(e); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	all, err := db.Query(history.HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(all))
	}
	id := all[0].ID

	got, err := db.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil entry")
	}
	if got.Filename != e.Filename {
		t.Errorf("Filename: got %q, want %q", got.Filename, e.Filename)
	}
	if got.PageCount != e.PageCount {
		t.Errorf("PageCount: got %d, want %d", got.PageCount, e.PageCount)
	}

	missing, err := db.GetByID(99999)
	if err != nil {
		t.Fatalf("GetByID missing: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestGetStats(t *testing.T) {
	db := openTestDB(t)

	// Insert today's entries
	toInsert := []history.HistoryEntry{
		{Filename: "p1.pdf", OriginalPath: "/p1.pdf", Status: "printed"},
		{Filename: "p2.pdf", OriginalPath: "/p2.pdf", Status: "printed"},
		{Filename: "p3.pdf", OriginalPath: "/p3.pdf", Status: "printed"},
		{Filename: "f1.pdf", OriginalPath: "/f1.pdf", Status: "failed"},
		{Filename: "f2.pdf", OriginalPath: "/f2.pdf", Status: "failed"},
		{Filename: "q1.pdf", OriginalPath: "/q1.pdf", Status: "pending_learning"},
		{Filename: "q2.pdf", OriginalPath: "/q2.pdf", Status: "pending_learning"},
	}
	for _, e := range toInsert {
		if err := db.Insert(e); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if stats.TodayPrinted != 3 {
		t.Errorf("TodayPrinted: got %d, want 3", stats.TodayPrinted)
	}
	if stats.TodayFailed != 2 {
		t.Errorf("TodayFailed: got %d, want 2", stats.TodayFailed)
	}
	if stats.PendingLearning != 2 {
		t.Errorf("PendingLearning: got %d, want 2", stats.PendingLearning)
	}
	if stats.TotalProcessed != 7 {
		t.Errorf("TotalProcessed: got %d, want 7", stats.TotalProcessed)
	}
}

func TestQueryDateRange(t *testing.T) {
	db := openTestDB(t)

	// Insert a few entries; use date filter to get none (past date range)
	if err := db.Insert(history.HistoryEntry{
		Filename:     "x.pdf",
		OriginalPath: "/x.pdf",
		Status:       "printed",
	}); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Filter with a future date range — should return 0 results
	future := "2099-01-01"
	results, err := db.Query(history.HistoryFilter{
		DateFrom: future,
		DateTo:   future,
		Limit:    50,
	})
	if err != nil {
		t.Fatalf("Query future range: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for future date range, got %d", len(results))
	}

	// Filter with today's date — should return the entry
	today := time.Now().Format("2006-01-02")
	results, err = db.Query(history.HistoryFilter{
		DateFrom: today,
		DateTo:   today,
		Limit:    50,
	})
	if err != nil {
		t.Fatalf("Query today range: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for today range, got %d", len(results))
	}
}

func TestPagination(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 10; i++ {
		if err := db.Insert(history.HistoryEntry{
			Filename:     "f.pdf",
			OriginalPath: "/f.pdf",
			Status:       "printed",
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	page1, err := db.Query(history.HistoryFilter{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("Query page1: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page1: expected 3, got %d", len(page1))
	}

	page2, err := db.Query(history.HistoryFilter{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("Query page2: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page2: expected 3, got %d", len(page2))
	}

	// IDs must be different across pages
	if page1[0].ID == page2[0].ID {
		t.Error("page1 and page2 should not share IDs")
	}
}

func TestGetDailyStats(t *testing.T) {
	db := openTestDB(t)
	db.Insert(history.HistoryEntry{Filename: "a.pdf", Status: "printed", OriginalPath: "/a"})
	db.Insert(history.HistoryEntry{Filename: "b.pdf", Status: "failed", OriginalPath: "/b"})
	db.Insert(history.HistoryEntry{Filename: "c.pdf", Status: "printed", OriginalPath: "/c"})

	stats, err := db.GetDailyStats(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 day, got %d", len(stats))
	}
	if stats[0].Printed != 2 || stats[0].Failed != 1 {
		t.Errorf("expected 2 printed, 1 failed, got %d/%d", stats[0].Printed, stats[0].Failed)
	}
}

func TestGetPrinterStats(t *testing.T) {
	db := openTestDB(t)
	db.Insert(history.HistoryEntry{Filename: "a.pdf", Status: "printed", Printer: "Zebra", PageCount: 1, OriginalPath: "/a"})
	db.Insert(history.HistoryEntry{Filename: "b.pdf", Status: "printed", Printer: "Zebra", PageCount: 2, OriginalPath: "/b"})
	db.Insert(history.HistoryEntry{Filename: "c.pdf", Status: "printed", Printer: "HP", PageCount: 3, OriginalPath: "/c"})

	stats, err := db.GetPrinterStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 printers, got %d", len(stats))
	}
}
