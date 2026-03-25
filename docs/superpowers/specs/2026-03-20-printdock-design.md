# PrintDock — Design Specification

## Overview

PrintDock is a Windows desktop application written in Go that watches a local folder for PDF files, classifies them using configurable rules, routes them to the correct printer silently, and archives them. When no rule matches, a learning popup lets the user choose a printer and optionally memorize the rule for future use.

Target: Windows 10/11 only. Offline-first. Single binary (+ embedded PDFtoPrinter.exe).

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go |
| UI framework | Wails v2 |
| Frontend | Vanilla HTML/CSS/JS |
| PDF analysis | pdfcpu (pure Go) |
| PDF preview | PDF.js (in frontend) |
| Silent printing | PDFtoPrinter.exe (embedded) |
| Database | modernc.org/sqlite (pure Go, no CGo) |
| File watching | fsnotify |
| Config/rules | YAML (gopkg.in/yaml.v3) |

## Architecture

Pipeline with goroutine workers. Linear flow: watch → process → classify → print → archive.

```
Watcher (fsnotify) → fileChan → Processor (2 workers)
                                    ├── Classifier match → Printer → Archiver (printed/)
                                    ├── No match → learningChan → UI queue
                                    ├── Learning "ignore" → Archiver (unknown/)
                                    └── Error → Archiver (failed/) + log
```

Learning queue is a separate channel. Files that match rules continue processing regardless of pending learning items. The learning popup stays visible until the user acts on every queued item.

## Project Structure

```
printdock/
├── main.go                  # Wails bootstrap
├── app.go                   # App struct, Wails bindings
├── internal/
│   ├── watcher/
│   │   └── watcher.go       # fsnotify + file stability check
│   ├── processor/
│   │   └── processor.go     # Pipeline orchestration
│   ├── classifier/
│   │   └── classifier.go    # Rule matching engine
│   ├── printer/
│   │   └── printer.go       # PDFtoPrinter.exe wrapper
│   ├── archiver/
│   │   └── archiver.go      # File move to dated dirs
│   ├── rules/
│   │   └── rules.go         # CRUD on rules.yaml
│   ├── history/
│   │   └── history.go       # SQLite operations
│   └── config/
│       └── config.go        # config.yaml load/save
├── frontend/
│   ├── index.html           # SPA shell, hash router
│   ├── style.css
│   ├── app.js               # Router + shared logic
│   ├── pages/
│   │   ├── dashboard.js
│   │   ├── settings.js
│   │   ├── history.js
│   │   └── learning.js
│   └── lib/
│       └── pdfjs/           # PDF.js library files
├── assets/
│   └── PDFtoPrinter.exe     # Embedded in binary via go:embed
├── config.yaml
├── rules.yaml
└── wails.json
```

## Data Models

### config.yaml

```yaml
watch_dir: "C:\\Users\\shop\\Downloads\\orders"
archive_dir: "C:\\Users\\shop\\PrintDock\\archive"
stability_delay_seconds: 2
worker_count: 2
```

### rules.yaml

```yaml
rules:
  - name: shipping-label
    match:
      filename_regex: "(?i)label|expedition|colissimo"
      page_width_mm: 100
      page_height_mm: 150
      dimension_tolerance_mm: 5
    action:
      printer: "Zebra_ZD421"
      archive_subdir: "thermal"

  - name: delivery-note
    match:
      filename_regex: "(?i)bl|bon[-_ ]?livraison"
      page_width_mm: 210
      page_height_mm: 297
      dimension_tolerance_mm: 5
    action:
      printer: "HP_LaserJet"
      archive_subdir: "a4"
```

### SQLite history table

```sql
CREATE TABLE history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    original_path TEXT NOT NULL,
    archive_path TEXT,
    printer TEXT,
    rule_name TEXT,
    status TEXT NOT NULL,  -- 'printed', 'failed', 'unknown', 'ignored'
    page_count INTEGER,
    page_width_mm REAL,
    page_height_mm REAL,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Go structs

```go
// Config represents the application configuration
type Config struct {
    WatchDir              string `yaml:"watch_dir"`
    ArchiveDir            string `yaml:"archive_dir"`
    StabilityDelaySeconds int    `yaml:"stability_delay_seconds"`
    WorkerCount           int    `yaml:"worker_count"`
}

// Rule represents a classification rule
type Rule struct {
    Name  string    `yaml:"name"`
    Match RuleMatch `yaml:"match"`
    Action RuleAction `yaml:"action"`
}

type RuleMatch struct {
    FilenameRegex      string  `yaml:"filename_regex,omitempty"`
    PageWidthMM        float64 `yaml:"page_width_mm,omitempty"`
    PageHeightMM       float64 `yaml:"page_height_mm,omitempty"`
    DimensionTolerance float64 `yaml:"dimension_tolerance_mm,omitempty"`
    PageCount          int     `yaml:"page_count,omitempty"`
}

type RuleAction struct {
    Printer       string `yaml:"printer"`
    ArchiveSubdir string `yaml:"archive_subdir"`
}

// LearningItem represents a file pending user decision
type LearningItem struct {
    ID           string  // UUID
    FilePath     string
    Filename     string
    PageCount    int
    PageWidthMM  float64
    PageHeightMM float64
    DetectedAt   time.Time
}

// HistoryEntry represents a processed document
type HistoryEntry struct {
    ID           int64
    Filename     string
    OriginalPath string
    ArchivePath  string
    Printer      string
    RuleName     string
    Status       string
    PageCount    int
    PageWidthMM  float64
    PageHeightMM float64
    ErrorMessage string
    CreatedAt    time.Time
}
```

## Module Details

### Watcher

- Uses `fsnotify` to watch the configured directory for `Create` events
- Filters: only `.pdf` files
- Stability check: after detection, waits `stability_delay_seconds`, then compares file size. If unchanged, the file is ready. If changed, resets the timer.
- Sends stable file paths to `fileChan`
- Handles watcher errors with logging and automatic restart (retry after 5s, max 10 retries with exponential backoff capped at 60s)

### Processor

- Reads from `fileChan` with a pool of `worker_count` goroutines
- For each file:
  1. Extract PDF metadata via `pdfcpu`: page count, page 1 dimensions (converted to mm)
  2. Pass to Classifier
  3. If match: call Printer, then Archiver, then History.Insert
  4. If no match: push to `learningChan`
  5. On any error: log, archive to `failed/`, History.Insert with error

### Classifier

- Loads rules from `rules.yaml` at startup. Rules module calls `Classifier.Reload()` after any successful `Save()`/`Add()`/`Delete()`
- Tests rules in order (first match wins)
- Match logic per rule:
  1. If `filename_regex` is set: test against filename (not path)
  2. If dimensions are set: compare with tolerance (±`dimension_tolerance_mm`, default 5mm)
  3. If `page_count` is set: exact match
  4. All specified conditions must match (AND logic)
- Returns matched `Rule` or nil

### Printer

- Embeds `PDFtoPrinter.exe` via `go:embed`, extracts to temp dir on first use. Cleaned up on app shutdown via `Close()` method.
- `Print(filePath, printerName string) error`
- Executes: `PDFtoPrinter.exe <filepath> "<printer_name>"`
- Timeout: 30 seconds
- Lists available printers via Windows API (`winspool.drv` EnumPrinters)

### Archiver

- Moves files to the archive directory with dated structure:
  ```
  archive/
    printed/<subdir>/YYYY/MM/DD/
    failed/YYYY/MM/DD/
    unknown/YYYY/MM/DD/
  ```
- Renames files: `YYYYMMDD_HHMMSS_<printer>_<original>.pdf`. For files without a printer (failed/unknown), substitutes `noprinter` for the printer field.
- On name collision: appends `_1`, `_2`, etc. before `.pdf`
- Creates directories as needed

### Rules (CRUD)

- `Load() ([]Rule, error)` — parse rules.yaml
- `Save(rules []Rule) error` — write rules.yaml
- `Add(rule Rule) error` — append and save
- `Delete(name string) error` — remove by name and save
- File lock during writes to prevent corruption

### History (SQLite)

- `Insert(entry HistoryEntry) error`
- `Query(filter HistoryFilter) ([]HistoryEntry, error)` — supports status filter, date range, pagination
- `GetStats() (DashboardStats, error)` — counts by status, today's count
- Note: `ReprintDocument` is orchestrated by `App` (reads archive_path from History, calls Printer.Print). History only stores/queries data.

### Config

- `Load() (Config, error)` — parse config.yaml, apply defaults
- `Save(cfg Config) error` — write config.yaml
- Defaults: stability_delay=2s, worker_count=2

## Wails Bindings (app.go)

Methods exposed to the frontend:

```go
// Dashboard
func (a *App) GetDashboardStats() DashboardStats
func (a *App) GetWatcherStatus() WatcherStatus

// Configuration
func (a *App) GetConfig() Config
func (a *App) SaveConfig(cfg Config) error
func (a *App) GetPrinters() []string

// Rules
func (a *App) GetRules() []Rule
func (a *App) SaveRule(rule Rule) error
func (a *App) DeleteRule(name string) error

// History
func (a *App) GetHistory(filter HistoryFilter) []HistoryEntry
func (a *App) ReprintDocument(id int64) error

// Learning
func (a *App) GetLearningQueue() []LearningItem
func (a *App) HandleLearning(id string, action LearningAction) error
```

### Additional Go structs (bindings)

```go
type LearningAction struct {
    Action   string `json:"action"`   // "print", "print_and_memorize", "ignore"
    Printer  string `json:"printer"`
    RuleName string `json:"ruleName,omitempty"` // only for print_and_memorize
}

type HistoryFilter struct {
    Status   string `json:"status,omitempty"`   // "printed", "failed", "unknown", "ignored", or "" for all
    DateFrom string `json:"dateFrom,omitempty"` // "YYYY-MM-DD"
    DateTo   string `json:"dateTo,omitempty"`   // "YYYY-MM-DD"
    Offset   int    `json:"offset"`
    Limit    int    `json:"limit"` // default 50
}

type DashboardStats struct {
    TodayPrinted  int  `json:"todayPrinted"`
    TodayFailed   int  `json:"todayFailed"`
    PendingLearning int `json:"pendingLearning"`
    TotalProcessed int  `json:"totalProcessed"`
    WatcherRunning bool `json:"watcherRunning"`
}

type WatcherStatus struct {
    Running  bool   `json:"running"`
    WatchDir string `json:"watchDir"`
    Error    string `json:"error,omitempty"`
}
```

### Directory picker binding

The frontend uses a Wails binding to open native directory dialogs:

```go
func (a *App) PickDirectory(title string) (string, error)
// Uses runtime.OpenDirectoryDialog() internally
```

Events emitted (Go → JS):
- `file:processed` — refresh dashboard counters
- `learning:new` — new item in queue, update badge
- `learning:resolved` — item processed, decrement badge
- `print:error` — show error toast

## Frontend Pages

### Dashboard (`#dashboard`)
- Watcher status indicator (running/stopped)
- Watched folder path
- Available printers list
- Counters: today printed, today failed, pending learning
- Auto-refresh via Wails events

### Settings (`#settings`)
- Watch directory picker (Wails native dialog)
- Archive directory picker
- Rules list with add/edit/delete
- Rule editor: name, filename regex, dimensions, page count, printer, archive subdir

### History (`#history`)
- Table: date, filename, printer, rule, status
- Filters: status, date range
- Reprint button per row

### Learning (`#learning`)
- Persistent view — stays visible as long as items are in queue
- Badge counter on nav
- For each item:
  - PDF preview (PDF.js, page 1)
  - Filename, dimensions, page count
  - Printer dropdown
  - Archive subdir input (default: `unsorted`, user can change)
  - Three buttons: Print / Print + Memorize / Ignore
- When "Print" (without memorize): archives to `printed/<user-chosen-subdir>/YYYY/MM/DD/` (defaults to `unsorted`)
- When "Print + Memorize": pre-fills a rule with `filename_regex` set to the escaped literal filename, `page_width_mm` and `page_height_mm` from detected values, `dimension_tolerance_mm` of 5. User can edit all fields (especially the regex to make it more general) before confirming.

## Archive Structure

```
archive/
  printed/
    thermal/
      2026/03/20/
        20260320_143022_Zebra_ZD421_label-order-1234.pdf
    a4/
      2026/03/20/
        20260320_143055_HP_LaserJet_bl-order-1234.pdf
  failed/
    2026/03/20/
      20260320_143110_unknown_corrupted-file.pdf
  unknown/
    2026/03/20/
      20260320_143200_ignored_mystery-document.pdf
```

## Error Handling

| Error | Action |
|-------|--------|
| Printer offline | Log, move to `failed/`, history with error, toast in UI |
| PDF corrupted (pdfcpu fails) | Log, move to `failed/`, history with error |
| File locked | Retry 3 times with 1s delay, then fail |
| PDFtoPrinter timeout | Log, move to `failed/`, history with error |
| rules.yaml parse error | Log, keep previous rules in memory, show warning in UI |
| Watch dir deleted | Stop watcher, show error in dashboard, wait for reconfiguration |
| Archive file missing on reprint | Return error with clear message, do not update history status |

## Concurrency

- `fileChan`: buffered channel (capacity 100)
- `learningChan`: buffered channel (capacity 50), drained by a dedicated goroutine into an in-memory slice on the App struct
- Workers: 2 goroutines reading from `fileChan`
- File deduplication: processor tracks in-flight files by path to prevent double processing
- `sync.Mutex` on rules.yaml writes and learning queue access

## Dependencies

```
github.com/wailsapp/wails/v2
github.com/fsnotify/fsnotify
github.com/pdfcpu/pdfcpu
modernc.org/sqlite
gopkg.in/yaml.v3
github.com/google/uuid
```

## Build

```bash
# Development
wails dev

# Production Windows binary
wails build -platform windows/amd64
```

Output: single `printdock.exe` with embedded frontend and PDFtoPrinter.exe.
