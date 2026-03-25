# PrintDock Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Windows desktop app that watches a folder for PDFs, classifies them via rules, prints silently, and archives — with a learning mode for unmatched files.

**Architecture:** Wails v2 app with Go backend pipeline (watcher → processor → classifier → printer → archiver) and vanilla JS frontend. All modules communicate via Go channels. Learning queue is separate from the main pipeline.

**Tech Stack:** Go, Wails v2, pdfcpu, PDF.js, PDFtoPrinter.exe, modernc.org/sqlite, fsnotify, gopkg.in/yaml.v3

**Spec:** `docs/superpowers/specs/2026-03-20-printdock-design.md`

**Note:** Development is on Linux, target is Windows. Modules that use Windows APIs (printer enumeration, PDFtoPrinter) use build tags. Core logic (config, rules, classifier, archiver, history) is cross-platform and fully testable on Linux.

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`, `wails.json`, `main.go`, `app.go`
- Create: `frontend/index.html`, `frontend/style.css`, `frontend/app.js`
- Create: `config.yaml`, `rules.yaml`
- Create: all `internal/*/` directories

- [ ] **Step 1: Initialize Wails project**

```bash
cd /home/ecohen/perso/morpheus/printdock
go mod init printdock
```

- [ ] **Step 2: Create wails.json**

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "PrintDock",
  "outputfilename": "printdock",
  "frontend:install": "",
  "frontend:build": "",
  "frontend:dev:watcher": "",
  "frontend:dev:serverUrl": "",
  "author": {
    "name": "PrintDock"
  }
}
```

- [ ] **Step 3: Create main.go with Wails bootstrap**

```go
package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "PrintDock",
		Width:  1024,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
```

- [ ] **Step 4: Create app.go with minimal App struct**

```go
package main

import "context"

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
}
```

- [ ] **Step 5: Create frontend shell**

`frontend/index.html`:
```html
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PrintDock</title>
    <link rel="stylesheet" href="/style.css">
</head>
<body>
    <nav id="nav">
        <div class="nav-brand">PrintDock</div>
        <div class="nav-links">
            <a href="#dashboard" class="nav-link active">Dashboard</a>
            <a href="#settings" class="nav-link">Configuration</a>
            <a href="#history" class="nav-link">Historique</a>
            <a href="#learning" class="nav-link">
                Apprentissage
                <span id="learning-badge" class="badge" style="display:none">0</span>
            </a>
        </div>
    </nav>
    <main id="app"></main>
    <div id="toast-container"></div>
    <script src="/app.js"></script>
    <script src="/pages/dashboard.js"></script>
    <script src="/pages/settings.js"></script>
    <script src="/pages/history.js"></script>
    <script src="/pages/learning.js"></script>
</body>
</html>
```

`frontend/app.js`:
```js
const routes = {};

function registerPage(hash, renderFn) {
    routes[hash] = renderFn;
}

function navigate() {
    const hash = window.location.hash || '#dashboard';
    const app = document.getElementById('app');
    document.querySelectorAll('.nav-link').forEach(l => {
        l.classList.toggle('active', l.getAttribute('href') === hash);
    });
    const render = routes[hash];
    if (render) {
        app.innerHTML = '';
        render(app);
    }
}

function showToast(message, type) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast toast-' + (type || 'info');
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 4000);
}

window.addEventListener('hashchange', navigate);
window.addEventListener('DOMContentLoaded', navigate);
```

- [ ] **Step 6: Create minimal CSS**

`frontend/style.css` — functional, clean styling for nav, pages, cards, tables, forms, toasts, badges.

- [ ] **Step 7: Create placeholder page JS files**

Each of `frontend/pages/dashboard.js`, `settings.js`, `history.js`, `learning.js`:
```js
registerPage('#dashboard', function(container) {
    container.innerHTML = '<h1>Dashboard</h1><p>En construction...</p>';
});
```

- [ ] **Step 8: Create default config.yaml and rules.yaml**

`config.yaml`:
```yaml
watch_dir: ""
archive_dir: ""
stability_delay_seconds: 2
worker_count: 2
```

`rules.yaml`:
```yaml
rules: []
```

- [ ] **Step 9: Create internal directories**

```bash
mkdir -p internal/{watcher,processor,classifier,printer,archiver,rules,history,config}
```

- [ ] **Step 10: Install Go dependencies**

```bash
go get github.com/wailsapp/wails/v2
go get github.com/fsnotify/fsnotify
go get github.com/pdfcpu/pdfcpu/pkg/api
go get modernc.org/sqlite
go get gopkg.in/yaml.v3
go get github.com/google/uuid
```

- [ ] **Step 11: Verify build compiles**

```bash
go build ./...
```

---

## Task 2: Config Module

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write tests for config load/save**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StabilityDelaySeconds != 2 {
		t.Errorf("expected default stability_delay=2, got %d", cfg.StabilityDelaySeconds)
	}
	if cfg.WorkerCount != 2 {
		t.Errorf("expected default worker_count=2, got %d", cfg.WorkerCount)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("watch_dir: /tmp/watch\narchive_dir: /tmp/archive\nstability_delay_seconds: 5\nworker_count: 4\n"), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WatchDir != "/tmp/watch" {
		t.Errorf("expected watch_dir=/tmp/watch, got %s", cfg.WatchDir)
	}
	if cfg.StabilityDelaySeconds != 5 {
		t.Errorf("expected stability_delay=5, got %d", cfg.StabilityDelaySeconds)
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{WatchDir: "/tmp/test", ArchiveDir: "/tmp/archive", StabilityDelaySeconds: 3, WorkerCount: 1}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.WatchDir != cfg.WatchDir {
		t.Errorf("expected %s, got %s", cfg.WatchDir, loaded.WatchDir)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 3: Implement config.go**

```go
package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	WatchDir              string `yaml:"watch_dir" json:"watchDir"`
	ArchiveDir            string `yaml:"archive_dir" json:"archiveDir"`
	StabilityDelaySeconds int    `yaml:"stability_delay_seconds" json:"stabilityDelaySeconds"`
	WorkerCount           int    `yaml:"worker_count" json:"workerCount"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		StabilityDelaySeconds: 2,
		WorkerCount:           2,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.StabilityDelaySeconds <= 0 {
		cfg.StabilityDelaySeconds = 2
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./internal/config/ -v
```

---

## Task 3: Rules Module

**Files:**
- Create: `internal/rules/rules.go`
- Test: `internal/rules/rules_test.go`

- [ ] **Step 1: Write tests**

```go
package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, []byte("rules: []\n"), 0644)

	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, []byte(`rules:
  - name: test-label
    match:
      filename_regex: "(?i)label"
      page_width_mm: 100
      page_height_mm: 150
      dimension_tolerance_mm: 5
    action:
      printer: "TestPrinter"
      archive_subdir: "thermal"
`), 0644)

	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "test-label" {
		t.Errorf("expected name=test-label, got %s", rules[0].Name)
	}
	if rules[0].Match.PageWidthMM != 100 {
		t.Errorf("expected width=100, got %f", rules[0].Match.PageWidthMM)
	}
}

func TestAddRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, []byte("rules: []\n"), 0644)

	store := NewStore(path)
	err := store.Add(Rule{Name: "new-rule", Match: RuleMatch{FilenameRegex: "test"}, Action: RuleAction{Printer: "P1", ArchiveSubdir: "a4"}})
	if err != nil {
		t.Fatal(err)
	}
	rules, _ := Load(path)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestDeleteRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	os.WriteFile(path, []byte(`rules:
  - name: keep
    match: {}
    action:
      printer: P1
      archive_subdir: a4
  - name: remove
    match: {}
    action:
      printer: P2
      archive_subdir: thermal
`), 0644)

	store := NewStore(path)
	err := store.Delete("remove")
	if err != nil {
		t.Fatal(err)
	}
	rules, _ := Load(path)
	if len(rules) != 1 || rules[0].Name != "keep" {
		t.Errorf("expected only 'keep' rule remaining")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

- [ ] **Step 3: Implement rules.go**

```go
package rules

import (
	"fmt"
	"os"
	"sync"
	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name   string     `yaml:"name" json:"name"`
	Match  RuleMatch  `yaml:"match" json:"match"`
	Action RuleAction `yaml:"action" json:"action"`
}

type RuleMatch struct {
	FilenameRegex      string  `yaml:"filename_regex,omitempty" json:"filenameRegex"`
	PageWidthMM        float64 `yaml:"page_width_mm,omitempty" json:"pageWidthMm"`
	PageHeightMM       float64 `yaml:"page_height_mm,omitempty" json:"pageHeightMm"`
	DimensionTolerance float64 `yaml:"dimension_tolerance_mm,omitempty" json:"dimensionToleranceMm"`
	PageCount          int     `yaml:"page_count,omitempty" json:"pageCount"`
}

type RuleAction struct {
	Printer       string `yaml:"printer" json:"printer"`
	ArchiveSubdir string `yaml:"archive_subdir" json:"archiveSubdir"`
}

type rulesFile struct {
	Rules []Rule `yaml:"rules"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func Load(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f rulesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Rules, nil
}

func save(path string, rulesList []Rule) error {
	f := rulesFile{Rules: rulesList}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Store) Load() ([]Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Load(s.path)
}

func (s *Store) Save(rulesList []Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return save(s.path, rulesList)
}

func (s *Store) Add(rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, err := Load(s.path)
	if err != nil {
		return err
	}
	for _, r := range existing {
		if r.Name == rule.Name {
			return fmt.Errorf("rule %q already exists", rule.Name)
		}
	}
	existing = append(existing, rule)
	return save(s.path, existing)
}

func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, err := Load(s.path)
	if err != nil {
		return err
	}
	filtered := make([]Rule, 0, len(existing))
	found := false
	for _, r := range existing {
		if r.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("rule %q not found", name)
	}
	return save(s.path, filtered)
}
```

- [ ] **Step 4: Run tests — verify they pass**

---

## Task 4: Classifier Module

**Files:**
- Create: `internal/classifier/classifier.go`
- Test: `internal/classifier/classifier_test.go`

- [ ] **Step 1: Write tests**

```go
package classifier

import (
	"printdock/internal/rules"
	"testing"
)

func TestMatchByFilename(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "label", Match: rules.RuleMatch{FilenameRegex: "(?i)label"}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	result := c.Classify("order-label-123.pdf", 100, 150, 1)
	if result == nil || result.Name != "label" {
		t.Errorf("expected match on label rule")
	}
}

func TestMatchByDimensions(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "a4", Match: rules.RuleMatch{PageWidthMM: 210, PageHeightMM: 297, DimensionTolerance: 5}, Action: rules.RuleAction{Printer: "HP"}},
	})
	result := c.Classify("random.pdf", 211, 296, 1)
	if result == nil || result.Name != "a4" {
		t.Errorf("expected match on a4 rule (within tolerance)")
	}
}

func TestNoMatch(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "label", Match: rules.RuleMatch{FilenameRegex: "(?i)label"}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	result := c.Classify("invoice.pdf", 210, 297, 1)
	if result != nil {
		t.Errorf("expected no match, got %s", result.Name)
	}
}

func TestDimensionOutOfTolerance(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "thermal", Match: rules.RuleMatch{PageWidthMM: 100, PageHeightMM: 150, DimensionTolerance: 5}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	result := c.Classify("file.pdf", 120, 150, 1)
	if result != nil {
		t.Errorf("expected no match (width 120 too far from 100)")
	}
}

func TestMatchByPageCount(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "multi", Match: rules.RuleMatch{PageCount: 3}, Action: rules.RuleAction{Printer: "HP"}},
	})
	result := c.Classify("doc.pdf", 210, 297, 3)
	if result == nil {
		t.Errorf("expected match on page count")
	}
	result = c.Classify("doc.pdf", 210, 297, 1)
	if result != nil {
		t.Errorf("expected no match (wrong page count)")
	}
}

func TestANDLogic(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "strict", Match: rules.RuleMatch{FilenameRegex: "(?i)label", PageWidthMM: 100, PageHeightMM: 150, DimensionTolerance: 5}, Action: rules.RuleAction{Printer: "Zebra"}},
	})
	// filename matches but dimensions don't
	result := c.Classify("label.pdf", 210, 297, 1)
	if result != nil {
		t.Errorf("expected no match (dimensions mismatch)")
	}
	// both match
	result = c.Classify("label.pdf", 100, 150, 1)
	if result == nil {
		t.Errorf("expected match")
	}
}

func TestFirstMatchWins(t *testing.T) {
	c := New([]rules.Rule{
		{Name: "first", Match: rules.RuleMatch{FilenameRegex: "(?i)doc"}, Action: rules.RuleAction{Printer: "P1"}},
		{Name: "second", Match: rules.RuleMatch{FilenameRegex: "(?i)doc"}, Action: rules.RuleAction{Printer: "P2"}},
	})
	result := c.Classify("doc.pdf", 0, 0, 1)
	if result == nil || result.Name != "first" {
		t.Errorf("expected first rule to win")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

- [ ] **Step 3: Implement classifier.go**

```go
package classifier

import (
	"math"
	"printdock/internal/rules"
	"regexp"
	"sync"
)

type Classifier struct {
	mu       sync.RWMutex
	rules    []rules.Rule
	compiled []*regexp.Regexp
}

func New(rulesList []rules.Rule) *Classifier {
	c := &Classifier{}
	c.loadRules(rulesList)
	return c
}

func (c *Classifier) loadRules(rulesList []rules.Rule) {
	compiled := make([]*regexp.Regexp, len(rulesList))
	for i, r := range rulesList {
		if r.Match.FilenameRegex != "" {
			re, err := regexp.Compile(r.Match.FilenameRegex)
			if err == nil {
				compiled[i] = re
			}
		}
	}
	c.rules = rulesList
	c.compiled = compiled
}

func (c *Classifier) Reload(rulesList []rules.Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadRules(rulesList)
}

func (c *Classifier) Classify(filename string, widthMM, heightMM float64, pageCount int) *rules.Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i, rule := range c.rules {
		if !c.matches(i, rule, filename, widthMM, heightMM, pageCount) {
			continue
		}
		r := rule
		return &r
	}
	return nil
}

func (c *Classifier) matches(idx int, rule rules.Rule, filename string, widthMM, heightMM float64, pageCount int) bool {
	m := rule.Match

	if m.FilenameRegex != "" {
		if c.compiled[idx] == nil || !c.compiled[idx].MatchString(filename) {
			return false
		}
	}

	tolerance := m.DimensionTolerance
	if tolerance <= 0 {
		tolerance = 5
	}

	if m.PageWidthMM > 0 {
		if math.Abs(widthMM-m.PageWidthMM) > tolerance {
			return false
		}
	}
	if m.PageHeightMM > 0 {
		if math.Abs(heightMM-m.PageHeightMM) > tolerance {
			return false
		}
	}

	if m.PageCount > 0 && pageCount != m.PageCount {
		return false
	}

	return true
}
```

- [ ] **Step 4: Run tests — verify they pass**

---

## Task 5: Archiver Module

**Files:**
- Create: `internal/archiver/archiver.go`
- Test: `internal/archiver/archiver_test.go`

- [ ] **Step 1: Write tests**

```go
package archiver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchivePrinted(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0755)
	srcFile := filepath.Join(src, "label.pdf")
	os.WriteFile(srcFile, []byte("fake pdf"), 0644)

	a := New(filepath.Join(dir, "archive"))
	path, err := a.Archive(srcFile, "printed", "thermal", "Zebra")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "printed") || !strings.Contains(path, "thermal") {
		t.Errorf("unexpected path: %s", path)
	}
	if !strings.Contains(path, "Zebra") {
		t.Errorf("expected printer name in path: %s", path)
	}
	// source should be gone
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("source file should be moved")
	}
}

func TestArchiveFailed(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "bad.pdf")
	os.WriteFile(srcFile, []byte("fake"), 0644)

	a := New(filepath.Join(dir, "archive"))
	path, err := a.Archive(srcFile, "failed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "failed") {
		t.Errorf("expected failed dir in path: %s", path)
	}
	if !strings.Contains(path, "noprinter") {
		t.Errorf("expected noprinter placeholder: %s", path)
	}
}

func TestArchiveCollision(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	a := New(archiveDir)

	// Create two files and archive them at the "same time"
	f1 := filepath.Join(dir, "doc.pdf")
	os.WriteFile(f1, []byte("a"), 0644)
	path1, _ := a.Archive(f1, "printed", "a4", "HP")

	f2 := filepath.Join(dir, "doc.pdf")
	os.WriteFile(f2, []byte("b"), 0644)
	path2, _ := a.Archive(f2, "printed", "a4", "HP")

	if path1 == path2 {
		t.Errorf("paths should differ on collision")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

- [ ] **Step 3: Implement archiver.go**

```go
package archiver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Archiver struct {
	baseDir string
}

func New(baseDir string) *Archiver {
	return &Archiver{baseDir: baseDir}
}

func (a *Archiver) Archive(srcPath, status, subdir, printer string) (string, error) {
	now := time.Now()
	dateDir := filepath.Join(fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()), fmt.Sprintf("%02d", now.Day()))

	var dir string
	switch status {
	case "printed":
		if subdir == "" {
			subdir = "unsorted"
		}
		dir = filepath.Join(a.baseDir, "printed", subdir, dateDir)
	case "failed":
		dir = filepath.Join(a.baseDir, "failed", dateDir)
	default:
		dir = filepath.Join(a.baseDir, "unknown", dateDir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if printer == "" {
		printer = "noprinter"
	}
	original := filepath.Base(srcPath)
	timestamp := now.Format("20060102_150405")
	newName := fmt.Sprintf("%s_%s_%s", timestamp, printer, original)
	destPath := filepath.Join(dir, newName)

	destPath = resolveCollision(destPath)

	if err := os.Rename(srcPath, destPath); err != nil {
		// fallback to copy+delete if rename fails (cross-device)
		data, readErr := os.ReadFile(srcPath)
		if readErr != nil {
			return "", readErr
		}
		if writeErr := os.WriteFile(destPath, data, 0644); writeErr != nil {
			return "", writeErr
		}
		os.Remove(srcPath)
	}
	return destPath, nil
}

func resolveCollision(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}
```

- [ ] **Step 4: Run tests — verify they pass**

---

## Task 6: History Module (SQLite)

**Files:**
- Create: `internal/history/history.go`
- Test: `internal/history/history_test.go`

- [ ] **Step 1: Write tests**

```go
package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInsertAndQuery(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	entry := HistoryEntry{
		Filename:     "label.pdf",
		OriginalPath: "/tmp/label.pdf",
		ArchivePath:  "/archive/printed/thermal/label.pdf",
		Printer:      "Zebra",
		RuleName:     "shipping-label",
		Status:       "printed",
		PageCount:    1,
		PageWidthMM:  100,
		PageHeightMM: 150,
	}
	if err := db.Insert(entry); err != nil {
		t.Fatal(err)
	}

	results, err := db.Query(HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Filename != "label.pdf" {
		t.Errorf("expected label.pdf, got %s", results[0].Filename)
	}
}

func TestQueryFilterByStatus(t *testing.T) {
	db, _ := Open(filepath.Join(t.TempDir(), "test.db"))
	defer db.Close()

	db.Insert(HistoryEntry{Filename: "a.pdf", Status: "printed", OriginalPath: "/a"})
	db.Insert(HistoryEntry{Filename: "b.pdf", Status: "failed", OriginalPath: "/b"})

	results, _ := db.Query(HistoryFilter{Status: "printed", Limit: 10})
	if len(results) != 1 || results[0].Filename != "a.pdf" {
		t.Errorf("filter by status failed")
	}
}

func TestGetStats(t *testing.T) {
	db, _ := Open(filepath.Join(t.TempDir(), "test.db"))
	defer db.Close()

	db.Insert(HistoryEntry{Filename: "a.pdf", Status: "printed", OriginalPath: "/a"})
	db.Insert(HistoryEntry{Filename: "b.pdf", Status: "printed", OriginalPath: "/b"})
	db.Insert(HistoryEntry{Filename: "c.pdf", Status: "failed", OriginalPath: "/c"})

	stats, err := db.GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TodayPrinted != 2 {
		t.Errorf("expected 2 printed today, got %d", stats.TodayPrinted)
	}
	if stats.TodayFailed != 1 {
		t.Errorf("expected 1 failed today, got %d", stats.TodayFailed)
	}
}

func TestGetEntryByID(t *testing.T) {
	db, _ := Open(filepath.Join(t.TempDir(), "test.db"))
	defer db.Close()

	db.Insert(HistoryEntry{Filename: "a.pdf", Status: "printed", OriginalPath: "/a", ArchivePath: "/archive/a.pdf"})

	results, _ := db.Query(HistoryFilter{Limit: 1})
	entry, err := db.GetByID(results[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Filename != "a.pdf" {
		t.Errorf("expected a.pdf, got %s", entry.Filename)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

- [ ] **Step 3: Implement history.go**

```go
package history

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

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

type HistoryFilter struct {
	Status   string `json:"status,omitempty"`
	DateFrom string `json:"dateFrom,omitempty"`
	DateTo   string `json:"dateTo,omitempty"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type DashboardStats struct {
	TodayPrinted    int  `json:"todayPrinted"`
	TodayFailed     int  `json:"todayFailed"`
	PendingLearning int  `json:"pendingLearning"`
	TotalProcessed  int  `json:"totalProcessed"`
	WatcherRunning  bool `json:"watcherRunning"`
}

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS history (
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
	)`); err != nil {
		return nil, err
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) Insert(e HistoryEntry) error {
	_, err := d.db.Exec(`INSERT INTO history (filename, original_path, archive_path, printer, rule_name, status, page_count, page_width_mm, page_height_mm, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Filename, e.OriginalPath, e.ArchivePath, e.Printer, e.RuleName, e.Status, e.PageCount, e.PageWidthMM, e.PageHeightMM, e.ErrorMessage)
	return err
}

func (d *DB) Query(f HistoryFilter) ([]HistoryEntry, error) {
	query := "SELECT id, filename, original_path, archive_path, printer, rule_name, status, page_count, page_width_mm, page_height_mm, error_message, created_at FROM history WHERE 1=1"
	args := []interface{}{}

	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.DateFrom != "" {
		query += " AND date(created_at) >= ?"
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		query += " AND date(created_at) <= ?"
		args = append(args, f.DateTo)
	}

	query += " ORDER BY created_at DESC"

	if f.Limit <= 0 {
		f.Limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var createdAt string
		if err := rows.Scan(&e.ID, &e.Filename, &e.OriginalPath, &e.ArchivePath, &e.Printer, &e.RuleName, &e.Status, &e.PageCount, &e.PageWidthMM, &e.PageHeightMM, &e.ErrorMessage, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		entries = append(entries, e)
	}
	return entries, nil
}

func (d *DB) GetByID(id int64) (*HistoryEntry, error) {
	var e HistoryEntry
	var createdAt string
	err := d.db.QueryRow("SELECT id, filename, original_path, archive_path, printer, rule_name, status, page_count, page_width_mm, page_height_mm, error_message, created_at FROM history WHERE id = ?", id).
		Scan(&e.ID, &e.Filename, &e.OriginalPath, &e.ArchivePath, &e.Printer, &e.RuleName, &e.Status, &e.PageCount, &e.PageWidthMM, &e.PageHeightMM, &e.ErrorMessage, &createdAt)
	if err != nil {
		return nil, err
	}
	e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &e, nil
}

func (d *DB) GetStats() (DashboardStats, error) {
	var stats DashboardStats
	today := time.Now().Format("2006-01-02")

	d.db.QueryRow("SELECT COUNT(*) FROM history WHERE status='printed' AND date(created_at)=?", today).Scan(&stats.TodayPrinted)
	d.db.QueryRow("SELECT COUNT(*) FROM history WHERE status='failed' AND date(created_at)=?", today).Scan(&stats.TodayFailed)
	d.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&stats.TotalProcessed)

	return stats, nil
}
```

- [ ] **Step 4: Run tests — verify they pass**

---

## Task 7: Printer Module (Windows-specific)

**Files:**
- Create: `internal/printer/printer.go` (interface + common logic)
- Create: `internal/printer/printer_windows.go` (Windows implementation)
- Create: `internal/printer/printer_stub.go` (Linux stub for dev)
- Test: `internal/printer/printer_test.go`

- [ ] **Step 1: Create printer interface and common types**

`internal/printer/printer.go`:
```go
package printer

type Printer interface {
	Print(filePath, printerName string) error
	ListPrinters() ([]string, error)
	Close() error
}
```

- [ ] **Step 2: Create Linux dev stub**

`internal/printer/printer_stub.go`:
```go
//go:build !windows

package printer

import "fmt"

type StubPrinter struct{}

func New() (Printer, error) {
	return &StubPrinter{}, nil
}

func (p *StubPrinter) Print(filePath, printerName string) error {
	fmt.Printf("[STUB] Would print %s to %s\n", filePath, printerName)
	return nil
}

func (p *StubPrinter) ListPrinters() ([]string, error) {
	return []string{"Zebra_ZD421_stub", "HP_LaserJet_stub"}, nil
}

func (p *StubPrinter) Close() error { return nil }
```

- [ ] **Step 3: Create Windows implementation**

`internal/printer/printer_windows.go`:
```go
//go:build windows

package printer

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

//go:embed assets/PDFtoPrinter.exe
var pdfToPrinterExe []byte

type WindowsPrinter struct {
	exePath string
}

func New() (Printer, error) {
	tmpDir := filepath.Join(os.TempDir(), "printdock")
	os.MkdirAll(tmpDir, 0755)
	exePath := filepath.Join(tmpDir, "PDFtoPrinter.exe")

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		if err := os.WriteFile(exePath, pdfToPrinterExe, 0755); err != nil {
			return nil, fmt.Errorf("failed to extract PDFtoPrinter.exe: %w", err)
		}
	}
	return &WindowsPrinter{exePath: exePath}, nil
}

func (p *WindowsPrinter) Print(filePath, printerName string) error {
	cmd := exec.Command(p.exePath, filePath, printerName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return fmt.Errorf("print timeout after 30s")
	}
}

func (p *WindowsPrinter) ListPrinters() ([]string, error) {
	dll := syscall.NewLazyDLL("winspool.drv")
	enumPrinters := dll.NewProc("EnumPrintersW")

	var needed, returned uint32
	enumPrinters.Call(2, 0, 2, 0, 0, uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&returned)))

	if needed == 0 {
		return nil, nil
	}

	buf := make([]byte, needed)
	ret, _, _ := enumPrinters.Call(2, 0, 2, uintptr(unsafe.Pointer(&buf[0])), uintptr(needed), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&returned)))
	if ret == 0 {
		return nil, fmt.Errorf("EnumPrinters failed")
	}

	type printerInfo2 struct {
		ServerName     *uint16
		PrinterName    *uint16
		ShareName      *uint16
		PortName       *uint16
		DriverName     *uint16
		Comment        *uint16
		Location       *uint16
		DevMode        uintptr
		SepFile        *uint16
		PrintProcessor *uint16
		Datatype       *uint16
		Parameters     *uint16
		SecurityDesc   uintptr
		Attributes     uint32
		Priority       uint32
		DefaultPriority uint32
		StartTime      uint32
		UntilTime      uint32
		Status         uint32
		Jobs           uint32
		AveragePPM     uint32
	}

	size := unsafe.Sizeof(printerInfo2{})
	printers := make([]string, 0, returned)
	for i := uint32(0); i < returned; i++ {
		info := (*printerInfo2)(unsafe.Pointer(&buf[uintptr(i)*size]))
		name := syscall.UTF16PtrToString(info.PrinterName)
		printers = append(printers, name)
	}
	return printers, nil
}

func (p *WindowsPrinter) Close() error {
	dir := filepath.Dir(p.exePath)
	return os.RemoveAll(dir)
}
```

- [ ] **Step 4: Verify build compiles on Linux**

```bash
go build ./internal/printer/
```

---

## Task 8: Watcher Module

**Files:**
- Create: `internal/watcher/watcher.go`
- Test: `internal/watcher/watcher_test.go`

- [ ] **Step 1: Write tests**

```go
package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectsNewPDF(t *testing.T) {
	dir := t.TempDir()
	fileChan := make(chan string, 10)
	w, err := New(dir, fileChan, 1)
	if err != nil {
		t.Fatal(err)
	}
	go w.Start()
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "test.pdf"), []byte("pdf content"), 0644)

	select {
	case path := <-fileChan:
		if filepath.Base(path) != "test.pdf" {
			t.Errorf("expected test.pdf, got %s", filepath.Base(path))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for file detection")
	}
}

func TestIgnoresNonPDF(t *testing.T) {
	dir := t.TempDir()
	fileChan := make(chan string, 10)
	w, _ := New(dir, fileChan, 1)
	go w.Start()
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("not a pdf"), 0644)

	select {
	case <-fileChan:
		t.Fatal("should not detect non-PDF files")
	case <-time.After(2 * time.Second):
		// expected
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

- [ ] **Step 3: Implement watcher.go**

```go
package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	dir            string
	fileChan       chan string
	stabilityDelay time.Duration
	fsWatcher      *fsnotify.Watcher
	stop           chan struct{}
	mu             sync.Mutex
	running        bool
}

func New(dir string, fileChan chan string, stabilityDelaySeconds int) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		dir:            dir,
		fileChan:       fileChan,
		stabilityDelay: time.Duration(stabilityDelaySeconds) * time.Second,
		fsWatcher:      fsw,
		stop:           make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() error {
	if err := w.fsWatcher.Add(w.dir); err != nil {
		return err
	}
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				if strings.ToLower(filepath.Ext(event.Name)) == ".pdf" {
					go w.waitForStability(event.Name)
				}
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("[watcher] error: %v", err)
		case <-w.stop:
			return nil
		}
	}
}

func (w *Watcher) waitForStability(path string) {
	var lastSize int64 = -1
	for {
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		size := info.Size()
		if size == lastSize {
			w.fileChan <- path
			return
		}
		lastSize = size
		time.Sleep(w.stabilityDelay)
	}
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.stop)
		w.fsWatcher.Close()
		w.running = false
	}
}

func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}
```

- [ ] **Step 4: Run tests — verify they pass**

---

## Task 9: PDF Metadata Extraction

**Files:**
- Create: `internal/processor/pdfinfo.go`
- Test: `internal/processor/pdfinfo_test.go`

- [ ] **Step 1: Write test**

We need a real small PDF for testing. Create a helper that generates a minimal valid PDF.

```go
package processor

import (
	"os"
	"path/filepath"
	"testing"
)

// Minimal valid PDF (1 page, A4)
var minimalPDF = []byte(`%PDF-1.0
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/MediaBox[0 0 595.28 841.89]/Parent 2 0 R/Resources<<>>>>endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
trailer<</Size 4/Root 1 0 R>>
startxref
226
%%EOF`)

func TestExtractPDFInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	os.WriteFile(path, minimalPDF, 0644)

	info, err := ExtractPDFInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.PageCount != 1 {
		t.Errorf("expected 1 page, got %d", info.PageCount)
	}
	// A4 = 210 x 297 mm (approximately, from 595.28 x 841.89 points)
	if info.WidthMM < 200 || info.WidthMM > 220 {
		t.Errorf("expected width ~210mm, got %.1f", info.WidthMM)
	}
	if info.HeightMM < 290 || info.HeightMM > 305 {
		t.Errorf("expected height ~297mm, got %.1f", info.HeightMM)
	}
}
```

- [ ] **Step 2: Run test — verify it fails**

- [ ] **Step 3: Implement pdfinfo.go**

```go
package processor

import (
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type PDFInfo struct {
	PageCount int
	WidthMM   float64
	HeightMM  float64
}

const pointsToMM = 25.4 / 72.0

func ExtractPDFInfo(path string) (*PDFInfo, error) {
	ctx, err := api.ReadContextFile(path)
	if err != nil {
		return nil, err
	}
	if err := api.OptimizeContext(ctx); err != nil {
		return nil, err
	}

	info := &PDFInfo{
		PageCount: ctx.PageCount,
	}

	if ctx.PageCount > 0 {
		dim, err := ctx.PageDims()
		if err != nil {
			return nil, err
		}
		if len(dim) > 0 {
			info.WidthMM = dim[0].Width * pointsToMM
			info.HeightMM = dim[0].Height * pointsToMM
		}
	}

	return info, nil
}
```

Note: The pdfcpu API may vary slightly — adjust imports based on the actual version installed. The key conversion is 1 point = 25.4/72 mm.

- [ ] **Step 4: Run test — verify it passes. Adjust pdfcpu API calls if needed.**

---

## Task 10: Processor Module (Pipeline Orchestration)

**Files:**
- Create: `internal/processor/processor.go`
- Test: `internal/processor/processor_test.go`

- [ ] **Step 1: Write tests**

```go
package processor

import (
	"os"
	"path/filepath"
	"printdock/internal/archiver"
	"printdock/internal/classifier"
	"printdock/internal/history"
	"printdock/internal/rules"
	"testing"
	"time"
)

type mockPrinter struct {
	printed []string
}

func (m *mockPrinter) Print(path, printer string) error {
	m.printed = append(m.printed, path)
	return nil
}
func (m *mockPrinter) ListPrinters() ([]string, error) { return nil, nil }
func (m *mockPrinter) Close() error                    { return nil }

func TestProcessFileWithMatch(t *testing.T) {
	dir := t.TempDir()
	watchDir := filepath.Join(dir, "watch")
	os.MkdirAll(watchDir, 0755)

	pdfPath := filepath.Join(watchDir, "label-test.pdf")
	os.WriteFile(pdfPath, minimalPDF, 0644)

	db, _ := history.Open(filepath.Join(dir, "history.db"))
	defer db.Close()
	arch := archiver.New(filepath.Join(dir, "archive"))
	mp := &mockPrinter{}

	rulesList := []rules.Rule{
		{Name: "label", Match: rules.RuleMatch{FilenameRegex: "(?i)label"}, Action: rules.RuleAction{Printer: "Zebra", ArchiveSubdir: "thermal"}},
	}
	cl := classifier.New(rulesList)

	fileChan := make(chan string, 10)
	learningChan := make(chan LearningItem, 50)

	p := New(fileChan, learningChan, cl, mp, arch, db, 1)
	go p.Start()

	fileChan <- pdfPath
	time.Sleep(2 * time.Second)
	p.Stop()

	if len(mp.printed) != 1 {
		t.Errorf("expected 1 print, got %d", len(mp.printed))
	}

	entries, _ := db.Query(history.HistoryFilter{Limit: 10})
	if len(entries) != 1 || entries[0].Status != "printed" {
		t.Errorf("expected 1 printed history entry")
	}
}

func TestProcessFileNoMatch(t *testing.T) {
	dir := t.TempDir()
	watchDir := filepath.Join(dir, "watch")
	os.MkdirAll(watchDir, 0755)

	pdfPath := filepath.Join(watchDir, "random.pdf")
	os.WriteFile(pdfPath, minimalPDF, 0644)

	db, _ := history.Open(filepath.Join(dir, "history.db"))
	defer db.Close()
	arch := archiver.New(filepath.Join(dir, "archive"))
	mp := &mockPrinter{}
	cl := classifier.New([]rules.Rule{})

	fileChan := make(chan string, 10)
	learningChan := make(chan LearningItem, 50)

	p := New(fileChan, learningChan, cl, mp, arch, db, 1)
	go p.Start()

	fileChan <- pdfPath
	time.Sleep(1 * time.Second)
	p.Stop()

	if len(mp.printed) != 0 {
		t.Errorf("expected 0 prints for unmatched file")
	}

	select {
	case item := <-learningChan:
		if item.Filename != "random.pdf" {
			t.Errorf("expected random.pdf in learning queue")
		}
	default:
		t.Error("expected item in learning channel")
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

- [ ] **Step 3: Implement processor.go**

```go
package processor

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"printdock/internal/archiver"
	"printdock/internal/classifier"
	"printdock/internal/history"
)

type PrinterInterface interface {
	Print(filePath, printerName string) error
	ListPrinters() ([]string, error)
	Close() error
}

type LearningItem struct {
	ID           string    `json:"id"`
	FilePath     string    `json:"filePath"`
	Filename     string    `json:"filename"`
	PageCount    int       `json:"pageCount"`
	PageWidthMM  float64   `json:"pageWidthMm"`
	PageHeightMM float64   `json:"pageHeightMm"`
	DetectedAt   time.Time `json:"detectedAt"`
}

type Processor struct {
	fileChan     chan string
	learningChan chan LearningItem
	classifier   *classifier.Classifier
	printer      PrinterInterface
	archiver     *archiver.Archiver
	history      *history.DB
	workerCount  int
	stop         chan struct{}
	wg           sync.WaitGroup
	inFlight     sync.Map
}

func New(fileChan chan string, learningChan chan LearningItem, cl *classifier.Classifier, pr PrinterInterface, ar *archiver.Archiver, h *history.DB, workers int) *Processor {
	return &Processor{
		fileChan:     fileChan,
		learningChan: learningChan,
		classifier:   cl,
		printer:      pr,
		archiver:     ar,
		history:      h,
		workerCount:  workers,
		stop:         make(chan struct{}),
	}
}

func (p *Processor) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Processor) Stop() {
	close(p.stop)
	p.wg.Wait()
}

func (p *Processor) worker() {
	defer p.wg.Done()
	for {
		select {
		case path, ok := <-p.fileChan:
			if !ok {
				return
			}
			if _, loaded := p.inFlight.LoadOrStore(path, true); loaded {
				continue
			}
			p.processFile(path)
			p.inFlight.Delete(path)
		case <-p.stop:
			return
		}
	}
}

func (p *Processor) processFile(path string) {
	filename := filepath.Base(path)
	log.Printf("[processor] processing %s", filename)

	info, err := ExtractPDFInfo(path)
	if err != nil {
		log.Printf("[processor] PDF extraction error: %v", err)
		archPath, _ := p.archiver.Archive(path, "failed", "", "")
		p.history.Insert(history.HistoryEntry{
			Filename:     filename,
			OriginalPath: path,
			ArchivePath:  archPath,
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		return
	}

	rule := p.classifier.Classify(filename, info.WidthMM, info.HeightMM, info.PageCount)
	if rule == nil {
		p.learningChan <- LearningItem{
			ID:           uuid.New().String(),
			FilePath:     path,
			Filename:     filename,
			PageCount:    info.PageCount,
			PageWidthMM:  info.WidthMM,
			PageHeightMM: info.HeightMM,
			DetectedAt:   time.Now(),
		}
		return
	}

	if err := p.printer.Print(path, rule.Action.Printer); err != nil {
		log.Printf("[processor] print error: %v", err)
		archPath, _ := p.archiver.Archive(path, "failed", "", rule.Action.Printer)
		p.history.Insert(history.HistoryEntry{
			Filename:     filename,
			OriginalPath: path,
			ArchivePath:  archPath,
			Printer:      rule.Action.Printer,
			RuleName:     rule.Name,
			Status:       "failed",
			ErrorMessage: err.Error(),
			PageCount:    info.PageCount,
			PageWidthMM:  info.WidthMM,
			PageHeightMM: info.HeightMM,
		})
		return
	}

	archPath, err := p.archiver.Archive(path, "printed", rule.Action.ArchiveSubdir, rule.Action.Printer)
	if err != nil {
		log.Printf("[processor] archive error: %v", err)
	}

	p.history.Insert(history.HistoryEntry{
		Filename:     filename,
		OriginalPath: path,
		ArchivePath:  archPath,
		Printer:      rule.Action.Printer,
		RuleName:     rule.Name,
		Status:       "printed",
		PageCount:    info.PageCount,
		PageWidthMM:  info.WidthMM,
		PageHeightMM: info.HeightMM,
	})
}
```

- [ ] **Step 4: Run tests — verify they pass**

---

## Task 11: App Struct — Full Wails Bindings

**Files:**
- Modify: `app.go`

- [ ] **Step 1: Implement full app.go with all bindings**

Wire all modules together: config, rules store, classifier, printer, archiver, history DB, watcher, processor. Implement all binding methods (GetDashboardStats, GetConfig, SaveConfig, GetPrinters, GetRules, SaveRule, DeleteRule, GetHistory, ReprintDocument, GetLearningQueue, HandleLearning, PickDirectory). Start the learning queue drain goroutine. Emit Wails events on file:processed, learning:new, learning:resolved, print:error.

Key wiring in `startup()`:
1. Load config
2. Open history DB
3. Init printer, archiver, classifier, rules store
4. Create channels (fileChan cap 100, learningChan cap 50)
5. Start learning queue drain goroutine
6. Start processor
7. Start watcher (if watch_dir configured)

Key wiring in `shutdown()`:
1. Stop watcher
2. Stop processor
3. Close history DB
4. Close printer (cleanup temp files)

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

---

## Task 12: Frontend — Dashboard Page

**Files:**
- Modify: `frontend/pages/dashboard.js`

- [ ] **Step 1: Implement dashboard**

Calls `window.go.main.App.GetDashboardStats()` and `GetWatcherStatus()` on load. Displays:
- Watcher status indicator (green dot running, red dot stopped, watched path)
- Stat cards: today printed, today failed, pending learning, total processed
- Available printers list via `GetPrinters()`
- Listens to Wails events `file:processed` to auto-refresh stats

---

## Task 13: Frontend — Settings Page

**Files:**
- Modify: `frontend/pages/settings.js`

- [ ] **Step 1: Implement settings**

Two sections:
1. **General config**: watch dir (text input + browse button calling `PickDirectory`), archive dir (same), save button calling `SaveConfig()`
2. **Rules management**: table listing rules from `GetRules()`, add/edit modal with fields (name, filename_regex, page_width_mm, page_height_mm, dimension_tolerance_mm, page_count, printer dropdown from `GetPrinters()`, archive_subdir), delete button calling `DeleteRule()`, save calls `SaveRule()`

---

## Task 14: Frontend — History Page

**Files:**
- Modify: `frontend/pages/history.js`

- [ ] **Step 1: Implement history**

Table with columns: date, filename, printer, rule, status (with color-coded badge).
Filter bar: status dropdown (all/printed/failed/unknown/ignored), date from, date to.
Reprint button per row calling `ReprintDocument(id)`.
Pagination: prev/next buttons, 50 per page.
Calls `GetHistory(filter)` on load and on filter change.

---

## Task 15: Frontend — Learning Page

**Files:**
- Modify: `frontend/pages/learning.js`

- [ ] **Step 1: Implement learning queue UI**

Displays pending items from `GetLearningQueue()`. For each item:
- PDF preview using PDF.js (`pdfjsLib.getDocument(filePath)` rendered to canvas)
- File info: filename, dimensions (W x H mm), page count
- Printer dropdown (from `GetPrinters()`)
- Archive subdir input (default: `unsorted`)
- Three action buttons:
  - **Imprimer** → `HandleLearning(id, {action:"print", printer, archiveSubdir})`
  - **Imprimer + Mémoriser** → opens rule editor pre-filled (escaped filename as regex, dimensions, tolerance 5), confirm → `HandleLearning(id, {action:"print_and_memorize", printer, ruleName, ...})`
  - **Ignorer** → `HandleLearning(id, {action:"ignore"})`

Badge counter in nav updated via `learning:new` and `learning:resolved` events.

- [ ] **Step 2: Download PDF.js and place in frontend/lib/pdfjs/**

```bash
# Download PDF.js pre-built release
curl -L https://github.com/nicbarker/pdfjs/releases/... -o frontend/lib/pdfjs/pdf.min.js
```

Use the latest stable PDF.js build from Mozilla CDN or npm. Need `pdf.min.js` and `pdf.worker.min.js`.

---

## Task 16: Frontend — CSS Styling

**Files:**
- Modify: `frontend/style.css`

- [ ] **Step 1: Write complete CSS**

Clean, functional design: dark nav sidebar or top nav, card-based dashboard, table styling for history, form styling for settings, learning cards with preview. Toast notifications. Badge counter styling. Status dots (green/red). Responsive basics.

---

## Task 17: Integration & Smoke Test

- [ ] **Step 1: Run `wails dev` and verify the app launches**

```bash
wails dev
```

- [ ] **Step 2: Test each page loads**

Navigate between dashboard, settings, history, learning. Verify no JS errors in console.

- [ ] **Step 3: Test config save/load**

Set watch_dir and archive_dir in settings, save, reload app, verify they persist.

- [ ] **Step 4: Test file detection**

Create a test rules.yaml with a rule. Drop a matching PDF in the watch folder. Verify it gets printed (stub) and archived correctly.

- [ ] **Step 5: Test learning flow**

Drop an unmatched PDF. Verify it appears in the learning page. Pick a printer, click "Print + Memorize". Verify rule is saved in rules.yaml.

- [ ] **Step 6: Test history**

Verify all processed files appear in history with correct status. Test status filter. Test reprint.

- [ ] **Step 7: Windows build**

```bash
wails build -platform windows/amd64
```

Verify `build/bin/printdock.exe` is produced.
