# PrintDock

Smart print routing agent for Windows. Watches a folder for PDFs, classifies them by filename, dimensions, or page count, and silently routes them to the right printer. Unknown documents trigger a learning popup to teach new rules. Built for e-commerce businesses printing labels and delivery notes.

## Features

- **Folder watching** — monitors a directory for new PDF files in real time
- **Auto-classification** — matches documents using configurable rules (filename regex, page dimensions, page count)
- **Silent printing** — routes PDFs to the correct printer via SumatraPDF with no popups
- **Learning mode** — when no rule matches, a popup lets you pick a printer and optionally memorize the rule
- **Multi-printer support** — thermal label printers, laser printers, any Windows printer
- **Auto-archiving** — organizes printed files into dated directories
- **History** — SQLite-backed log of all processed documents with search, filters, and reprint
- **System tray** — runs quietly in the background, pops up when attention is needed
- **Single instance** — launching a second instance brings the existing window to focus
- **Startup** — can be configured to start with Windows

## Screenshots

*Coming soon*

## Architecture

```
Watcher (fsnotify) --> Processor (workers)
                          |-- Rule match --> SumatraPDF --> Archive (printed/)
                          |-- No match  --> Learning queue --> UI popup
                          '-- Error     --> Archive (failed/)
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go |
| UI | [Wails v2](https://wails.io) + Vanilla HTML/CSS/JS |
| PDF analysis | [pdfcpu](https://github.com/pdfcpu/pdfcpu) (pure Go) |
| PDF preview | [PDF.js](https://mozilla.github.io/pdf.js/) |
| Silent printing | [SumatraPDF](https://www.sumatrapdfreader.org/) CLI |
| Database | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGo) |
| File watching | [fsnotify](https://github.com/fsnotify/fsnotify) |
| Config | YAML |

## Requirements

- Windows 10 or 11
- [SumatraPDF](https://www.sumatrapdfreader.org/) — PrintDock can download it automatically from the settings page

## Installation

Download the latest `printdock.exe` from the [Releases](https://github.com/printdock/printdock/releases) page and run it. No installation required — it's a single executable.

## Configuration

On first launch, open **Settings** and configure:

1. **Watch directory** — the folder where your PDFs land (e.g. your browser's download folder)
2. **Archive directory** — where processed files are stored
3. **Printers** — restrict which printers are available to users

### Rules

Rules determine how documents are routed. Each rule can match by:

- **Filename regex** — e.g. `(?i)label|colissimo` for shipping labels
- **Page dimensions** — e.g. 100x150mm for thermal labels, 210x297mm for A4
- **Page count** — exact match

Rules are evaluated in order — first match wins. You can reorder them in the settings.

Example `rules.yaml`:

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

## Building from source

Prerequisites:
- [Go 1.21+](https://go.dev/dl/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Development mode (hot reload)
wails dev

# Production build
wails build -platform windows/amd64
```

The output binary is at `build/bin/printdock.exe`.

## Project Structure

```
printdock/
  main.go                    # Wails bootstrap, single instance
  app.go                     # App struct, all Wails bindings
  tray_windows.go            # System tray (Win32 API)
  internal/
    watcher/                 # fsnotify + file stability check
    processor/               # Pipeline: classify -> print -> archive
    classifier/              # Rule matching engine
    printer/                 # SumatraPDF wrapper + Win32 printer enumeration
    archiver/                # File move/copy to dated directories
    rules/                   # CRUD on rules.yaml
    history/                 # SQLite operations
    config/                  # config.yaml load/save
    applog/                  # In-memory ring buffer for UI logs
  frontend/
    index.html               # SPA shell with hash router
    pages/
      dashboard.js           # Stats, printer status, daily chart
      settings.js            # Config, rules CRUD, maintenance
      history.js             # Filterable table, batch reprint, preview
      learning.js            # Learning queue with drag & drop
      logs.js                # Real-time application logs
```

## Co-authored

This project was co-authored with [Claude Code](https://claude.ai/claude-code) by Anthropic.

## License

MIT
