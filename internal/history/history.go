package history

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// HistoryEntry represents a single print job record.
type HistoryEntry struct {
	ID           int64     `json:"id"`
	Filename     string    `json:"filename"`
	OriginalPath string    `json:"originalPath"`
	ArchivePath  string    `json:"archivePath"`
	Printer      string    `json:"printer"`
	RuleName     string    `json:"ruleName"`
	Status       string    `json:"status"`
	PageCount    int       `json:"pageCount"`
	PageWidthMM  float64   `json:"pageWidthMm"`
	PageHeightMM float64   `json:"pageHeightMm"`
	ErrorMessage string    `json:"errorMessage"`
	CreatedAt    time.Time `json:"createdAt"`
}

// HistoryFilter defines query filter/pagination parameters.
type HistoryFilter struct {
	Status   string `json:"status,omitempty"`
	DateFrom string `json:"dateFrom,omitempty"`
	DateTo   string `json:"dateTo,omitempty"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

// DashboardStats holds aggregated counts for the dashboard.
type DashboardStats struct {
	TodayPrinted    int  `json:"todayPrinted"`
	TodayFailed     int  `json:"todayFailed"`
	PendingLearning int  `json:"pendingLearning"`
	TotalProcessed  int  `json:"totalProcessed"`
	WatcherRunning  bool `json:"watcherRunning"`
}

const schema = `
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    original_path TEXT NOT NULL,
    archive_path TEXT DEFAULT '',
    printer TEXT DEFAULT '',
    rule_name TEXT DEFAULT '',
    status TEXT NOT NULL,
    page_count INTEGER DEFAULT 0,
    page_width_mm REAL DEFAULT 0,
    page_height_mm REAL DEFAULT 0,
    error_message TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

// DB wraps the SQLite database connection.
type DB struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path and runs migrations.
func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("history.Open: %w", err)
	}
	// Set busy timeout to avoid SQLITE_BUSY on concurrent access
	if _, err := sqldb.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("history.Open pragma: %w", err)
	}
	if _, err := sqldb.Exec(schema); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("history.Open schema: %w", err)
	}
	return &DB{db: sqldb}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Insert adds a new HistoryEntry to the database.
func (d *DB) Insert(e HistoryEntry) error {
	const q = `
INSERT INTO history
    (filename, original_path, archive_path, printer, rule_name, status,
     page_count, page_width_mm, page_height_mm, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(q,
		e.Filename,
		e.OriginalPath,
		e.ArchivePath,
		e.Printer,
		e.RuleName,
		e.Status,
		e.PageCount,
		e.PageWidthMM,
		e.PageHeightMM,
		e.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("history.Insert: %w", err)
	}
	return nil
}

// Query retrieves history entries matching the given filter.
// If Limit is 0 it defaults to 50. Results are ordered by created_at DESC.
func (d *DB) Query(f HistoryFilter) ([]HistoryEntry, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	q := `SELECT id, filename, original_path, archive_path, printer, rule_name,
               status, page_count, page_width_mm, page_height_mm, error_message, created_at
          FROM history
          WHERE 1=1`
	args := []any{}

	if f.Status != "" {
		q += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.DateFrom != "" {
		q += " AND DATE(created_at) >= DATE(?)"
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		q += " AND DATE(created_at) <= DATE(?)"
		args = append(args, f.DateTo)
	}

	q += " ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("history.Query: %w", err)
	}
	defer rows.Close()

	var results []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var createdAt string
		if err := rows.Scan(
			&e.ID, &e.Filename, &e.OriginalPath, &e.ArchivePath,
			&e.Printer, &e.RuleName, &e.Status, &e.PageCount,
			&e.PageWidthMM, &e.PageHeightMM, &e.ErrorMessage, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("history.Query scan: %w", err)
		}
		e.CreatedAt, _ = parseTime(createdAt)
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history.Query rows: %w", err)
	}
	return results, nil
}

// GetByID returns the entry with the given ID, or nil if not found.
func (d *DB) GetByID(id int64) (*HistoryEntry, error) {
	const q = `SELECT id, filename, original_path, archive_path, printer, rule_name,
               status, page_count, page_width_mm, page_height_mm, error_message, created_at
          FROM history WHERE id = ?`
	row := d.db.QueryRow(q, id)
	var e HistoryEntry
	var createdAt string
	err := row.Scan(
		&e.ID, &e.Filename, &e.OriginalPath, &e.ArchivePath,
		&e.Printer, &e.RuleName, &e.Status, &e.PageCount,
		&e.PageWidthMM, &e.PageHeightMM, &e.ErrorMessage, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("history.GetByID: %w", err)
	}
	e.CreatedAt, _ = parseTime(createdAt)
	return &e, nil
}

// GetStats returns aggregated dashboard statistics.
func (d *DB) GetStats() (DashboardStats, error) {
	var stats DashboardStats
	today := time.Now().Format("2006-01-02")

	row := d.db.QueryRow(
		`SELECT COUNT(*) FROM history WHERE status = 'printed' AND DATE(created_at) = DATE(?)`,
		today,
	)
	if err := row.Scan(&stats.TodayPrinted); err != nil {
		return stats, fmt.Errorf("history.GetStats TodayPrinted: %w", err)
	}

	row = d.db.QueryRow(
		`SELECT COUNT(*) FROM history WHERE status = 'failed' AND DATE(created_at) = DATE(?)`,
		today,
	)
	if err := row.Scan(&stats.TodayFailed); err != nil {
		return stats, fmt.Errorf("history.GetStats TodayFailed: %w", err)
	}

	row = d.db.QueryRow(
		`SELECT COUNT(*) FROM history WHERE status = 'pending_learning'`,
	)
	if err := row.Scan(&stats.PendingLearning); err != nil {
		return stats, fmt.Errorf("history.GetStats PendingLearning: %w", err)
	}

	row = d.db.QueryRow(`SELECT COUNT(*) FROM history`)
	if err := row.Scan(&stats.TotalProcessed); err != nil {
		return stats, fmt.Errorf("history.GetStats TotalProcessed: %w", err)
	}

	return stats, nil
}

// DeleteOlderThan removes history entries older than the given number of days
// and returns the archive paths of the deleted entries so files can be cleaned up.
func (d *DB) DeleteOlderThan(days int) ([]string, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := d.db.Query("SELECT archive_path FROM history WHERE date(created_at) < date(?)", cutoff)
	if err != nil {
		return nil, fmt.Errorf("history.DeleteOlderThan query: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil && p != "" {
			paths = append(paths, p)
		}
	}

	_, err = d.db.Exec("DELETE FROM history WHERE date(created_at) < date(?)", cutoff)
	if err != nil {
		return nil, fmt.Errorf("history.DeleteOlderThan delete: %w", err)
	}
	return paths, nil
}

// DailyStats represents print counts for a single day.
type DailyStats struct {
	Date    string `json:"date"`    // "YYYY-MM-DD"
	Printed int    `json:"printed"`
	Failed  int    `json:"failed"`
}

// PrinterStats represents print counts for a single printer.
type PrinterStats struct {
	Printer string `json:"printer"`
	Count   int    `json:"count"`
	Pages   int    `json:"pages"`
}

// GetDailyStats returns daily print/fail counts for the last N days.
func (d *DB) GetDailyStats(days int) ([]DailyStats, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := d.db.Query(`
        SELECT date(created_at) as day,
               SUM(CASE WHEN status='printed' THEN 1 ELSE 0 END) as printed,
               SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END) as failed
        FROM history
        WHERE date(created_at) >= date(?)
        GROUP BY day
        ORDER BY day ASC`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("history.GetDailyStats: %w", err)
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		if err := rows.Scan(&s.Date, &s.Printed, &s.Failed); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetPrinterStats returns per-printer counts for today.
func (d *DB) GetPrinterStats() ([]PrinterStats, error) {
	today := time.Now().Format("2006-01-02")
	rows, err := d.db.Query(`
        SELECT printer, COUNT(*) as count, COALESCE(SUM(page_count), 0) as pages
        FROM history
        WHERE status='printed' AND date(created_at) = date(?) AND printer != ''
        GROUP BY printer
        ORDER BY count DESC`, today)
	if err != nil {
		return nil, fmt.Errorf("history.GetPrinterStats: %w", err)
	}
	defer rows.Close()

	var stats []PrinterStats
	for rows.Next() {
		var s PrinterStats
		if err := rows.Scan(&s.Printer, &s.Count, &s.Pages); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// Reset deletes all rows from the history table.
func (d *DB) Reset() error {
	_, err := d.db.Exec("DELETE FROM history")
	if err != nil {
		return fmt.Errorf("history.Reset: %w", err)
	}
	return nil
}

// parseTime attempts to parse an SQLite datetime string into time.Time.
func parseTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}
