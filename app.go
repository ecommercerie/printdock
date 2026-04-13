package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"net/http"
	"strings"

	"printdock/internal/applog"
	"printdock/internal/archiver"
	"printdock/internal/classifier"
	"printdock/internal/config"
	"printdock/internal/history"
	"printdock/internal/printer"
	"printdock/internal/processor"
	"printdock/internal/rules"
	"printdock/internal/watcher"
)

// WatcherStatus holds the current state of the file watcher.
type WatcherStatus struct {
	Running  bool   `json:"running"`
	WatchDir string `json:"watchDir"`
	Error    string `json:"error,omitempty"`
}

// LearningAction describes what to do with a pending learning item.
type LearningAction struct {
	Action        string `json:"action"`                    // "print", "print_and_memorize", "ignore"
	Printer       string `json:"printer"`
	RuleName      string `json:"ruleName,omitempty"`
	FilenameRegex string `json:"filenameRegex,omitempty"`   // regex pattern for the rule
	ArchiveSubdir string `json:"archiveSubdir,omitempty"`
}

// App is the main application struct wired to all internal modules.
type App struct {
	ctx context.Context

	// Paths
	configPath string
	rulesPath  string

	// Core modules
	cfg        config.Config
	rulesStore *rules.Store
	cl         *classifier.Classifier
	pr         printer.Printer
	ar         *archiver.Archiver
	hist       *history.DB
	wtch       *watcher.Watcher
	proc       *processor.Processor

	// Channels
	fileChan     chan string
	learningChan chan processor.LearningItem

	// Learning queue
	learningMu    sync.Mutex
	learningQueue []processor.LearningItem

	// App logger
	appLog *applog.Logger

	// Pause state
	paused bool

	// Lifecycle management
	learningStop chan struct{}
	cleanupStop  chan struct{}
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 1. Determine data directory (%APPDATA%\PrintDock on Windows, ~/.printdock on Linux)
	dataDir := getDataDir()
	os.MkdirAll(dataDir, 0755)

	// 2. Set paths
	a.configPath = filepath.Join(dataDir, "config.yaml")
	a.rulesPath = filepath.Join(dataDir, "rules.yaml")

	// 2b. Init app logger
	a.appLog = applog.New(500)
	a.appLog.Info("PrintDock starting, data dir: %s", dataDir)

	// 3. Load config
	cfg, err := config.Load(a.configPath)
	if err != nil {
		log.Printf("[app] warning: could not load config: %v", err)
	}
	a.cfg = cfg

	// 4. Open history DB
	histPath := filepath.Join(dataDir, "history.db")
	hist, err := history.Open(histPath)
	if err != nil {
		log.Printf("[app] warning: could not open history DB: %v", err)
	}
	a.hist = hist

	// 5. Init printer
	pr, err := printer.New()
	if err != nil {
		log.Printf("[app] warning: could not init printer: %v", err)
	}
	a.pr = pr

	// 6. Init archiver
	archiveDir := a.cfg.ArchiveDir
	if archiveDir == "" {
		archiveDir = filepath.Join(dataDir, "archive")
	}
	a.ar = archiver.New(archiveDir)
	log.Printf("[app] archive directory: %s", archiveDir)

	// 7. Load rules
	rulesList, err := rules.Load(a.rulesPath)
	if err != nil {
		log.Printf("[app] warning: could not load rules: %v", err)
		rulesList = []rules.Rule{}
	}

	// 8. Init classifier
	a.cl = classifier.New(rulesList)

	// 9. Init rules store
	a.rulesStore = rules.NewStore(a.rulesPath)

	// 10. Create channels
	a.fileChan = make(chan string, 100)
	a.learningChan = make(chan processor.LearningItem, 50)
	a.learningStop = make(chan struct{})

	// 11. Start learning queue drain goroutine
	go a.drainLearningChan()

	// 12. Start processor
	workers := a.cfg.WorkerCount
	if workers <= 0 {
		workers = 2
	}
	a.proc = processor.New(a.fileChan, a.learningChan, a.cl, a.pr, a.ar, a.hist, workers, a.appLog)
	a.proc.Start()

	// 12b. Apply keep-original setting
	a.ar.SetKeepOriginal(a.cfg.KeepOriginal)

	// 13. If watch_dir is configured, start watcher
	if a.cfg.WatchDir != "" {
		a.startWatcher(a.cfg.WatchDir)
	}

	// 13b. Rescan existing files if configured
	if a.cfg.RescanOnStartup && a.wtch != nil {
		go func() {
			count, err := a.wtch.ScanExisting()
			if err != nil {
				a.appLog.Warn("Rescan error: %v", err)
			} else if count > 0 {
				a.appLog.Info("Rescan: %d fichier(s) existant(s) envoyé(s) au traitement", count)
			}
		}()
	}

	// 13c. Start extra watchers
	for _, dir := range a.cfg.ExtraWatchDirs {
		if dir != "" {
			a.startExtraWatcher(dir)
			a.appLog.Info("Extra watcher started: %s", dir)
		}
	}

	// 14. Start auto-cleanup goroutine
	a.cleanupStop = make(chan struct{})
	go a.autoCleanupLoop()

	// 15. Init system tray
	a.initTray()
}

// startWatcher creates and starts the file watcher for the given directory.
func (a *App) startWatcher(dir string) {
	delay := a.cfg.StabilityDelaySeconds
	if delay <= 0 {
		delay = 2
	}
	w, err := watcher.New(dir, a.fileChan, delay)
	if err != nil {
		log.Printf("[app] watcher init error: %v", err)
		return
	}
	a.wtch = w
	go func() {
		if err := w.Start(); err != nil {
			log.Printf("[app] watcher stopped with error: %v", err)
		}
	}()
}

// stopWatcher stops the current watcher if running.
func (a *App) stopWatcher() {
	if a.wtch != nil {
		a.wtch.Stop()
		a.wtch = nil
	}
}

// drainLearningChan reads from learningChan and appends to the in-memory queue.
func (a *App) drainLearningChan() {
	for {
		select {
		case item, ok := <-a.learningChan:
			if !ok {
				return
			}
			a.learningMu.Lock()
			a.learningQueue = append(a.learningQueue, item)
			a.learningMu.Unlock()
			runtime.EventsEmit(a.ctx, "learning:new", item)
			// Show window, bring to focus, and navigate to learning page
			a.ShowWindow()
			runtime.EventsEmit(a.ctx, "navigate:learning")
		case <-a.learningStop:
			return
		}
	}
}

// shutdown is called by Wails when the application is closing.
func (a *App) shutdown(ctx context.Context) {
	// 1. Stop watcher
	a.stopWatcher()

	// 2. Stop learning drain goroutine
	if a.learningStop != nil {
		close(a.learningStop)
	}

	// 2b. Stop cleanup goroutine
	if a.cleanupStop != nil {
		close(a.cleanupStop)
	}

	// 3. Stop processor
	if a.proc != nil {
		a.proc.Stop()
	}

	// 4. Close history DB
	if a.hist != nil {
		if err := a.hist.Close(); err != nil {
			log.Printf("[app] error closing history DB: %v", err)
		}
	}

	// 5. Close printer
	if a.pr != nil {
		if err := a.pr.Close(); err != nil {
			log.Printf("[app] error closing printer: %v", err)
		}
	}
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

// GetDashboardStats returns aggregated statistics for the dashboard.
func (a *App) GetDashboardStats() history.DashboardStats {
	if a.hist == nil {
		return history.DashboardStats{}
	}
	stats, err := a.hist.GetStats()
	if err != nil {
		log.Printf("[app] GetDashboardStats error: %v", err)
		return history.DashboardStats{}
	}
	stats.WatcherRunning = a.wtch != nil && a.wtch.IsRunning()

	a.learningMu.Lock()
	stats.PendingLearning = len(a.learningQueue)
	a.learningMu.Unlock()

	return stats
}

// GetWatcherStatus returns the current status of the file watcher.
func (a *App) GetWatcherStatus() WatcherStatus {
	if a.wtch == nil {
		return WatcherStatus{
			Running:  false,
			WatchDir: a.cfg.WatchDir,
		}
	}
	return WatcherStatus{
		Running:  a.wtch.IsRunning(),
		WatchDir: a.cfg.WatchDir,
	}
}

// ─── Config ──────────────────────────────────────────────────────────────────

// GetConfig returns the current configuration.
func (a *App) GetConfig() config.Config {
	return a.cfg
}

// SaveConfig persists the configuration and restarts the watcher if watch_dir changed.
func (a *App) SaveConfig(cfg config.Config) error {
	oldWatchDir := a.cfg.WatchDir

	if err := config.Save(a.configPath, cfg); err != nil {
		return fmt.Errorf("SaveConfig: %w", err)
	}
	a.cfg = cfg

	// Update archiver if archive_dir changed
	if cfg.ArchiveDir != "" {
		a.ar = archiver.New(cfg.ArchiveDir)
	}
	a.ar.SetKeepOriginal(cfg.KeepOriginal)

	if cfg.WatchDir != oldWatchDir {
		a.stopWatcher()
		if cfg.WatchDir != "" {
			a.startWatcher(cfg.WatchDir)
		}
	}
	return nil
}

// GetPrinters returns the list of available printer names, filtered by AllowedPrinters if set.
func (a *App) GetPrinters() []string {
	if a.pr == nil {
		return []string{}
	}
	printers, err := a.pr.ListPrinters()
	if err != nil {
		log.Printf("[app] GetPrinters error: %v", err)
		return []string{}
	}
	if len(a.cfg.AllowedPrinters) == 0 {
		return printers
	}
	allowed := make(map[string]bool, len(a.cfg.AllowedPrinters))
	for _, p := range a.cfg.AllowedPrinters {
		allowed[p] = true
	}
	filtered := make([]string, 0)
	for _, p := range printers {
		if allowed[p] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// GetAllPrinters returns ALL printers without filtering (for config page).
func (a *App) GetAllPrinters() []string {
	if a.pr == nil {
		return []string{}
	}
	printers, err := a.pr.ListPrinters()
	if err != nil {
		return []string{}
	}
	return printers
}

// ─── Rules ───────────────────────────────────────────────────────────────────

// GetRules returns the current list of routing rules.
func (a *App) GetRules() []rules.Rule {
	if a.rulesStore == nil {
		return []rules.Rule{}
	}
	rulesList, err := a.rulesStore.Load()
	if err != nil {
		log.Printf("[app] GetRules error: %v", err)
		return []rules.Rule{}
	}
	return rulesList
}

// SaveRule adds a new rule to the store and reloads the classifier.
func (a *App) SaveRule(rule rules.Rule) error {
	if err := a.rulesStore.Add(rule); err != nil {
		return fmt.Errorf("SaveRule: %w", err)
	}
	return a.reloadClassifier()
}

// DeleteRule removes a rule from the store and reloads the classifier.
func (a *App) DeleteRule(name string) error {
	if err := a.rulesStore.Delete(name); err != nil {
		return fmt.Errorf("DeleteRule: %w", err)
	}
	return a.reloadClassifier()
}

// reloadClassifier reads the current rules from disk and reloads the classifier.
func (a *App) reloadClassifier() error {
	rulesList, err := a.rulesStore.Load()
	if err != nil {
		return fmt.Errorf("reloadClassifier: %w", err)
	}
	a.cl.Reload(rulesList)
	return nil
}

// ─── History ─────────────────────────────────────────────────────────────────

// GetHistory returns history entries matching the given filter.
func (a *App) GetHistory(filter history.HistoryFilter) []history.HistoryEntry {
	if a.hist == nil {
		return []history.HistoryEntry{}
	}
	entries, err := a.hist.Query(filter)
	if err != nil {
		log.Printf("[app] GetHistory error: %v", err)
		return []history.HistoryEntry{}
	}
	if entries == nil {
		return []history.HistoryEntry{}
	}
	return entries
}

// ReprintDocument retrieves a history entry by ID and re-sends the archived file to the printer.
func (a *App) ReprintDocument(id int64) error {
	if a.hist == nil {
		return fmt.Errorf("history database not available")
	}
	entry, err := a.hist.GetByID(id)
	if err != nil {
		return fmt.Errorf("ReprintDocument: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("ReprintDocument: document %d not found", id)
	}
	if a.pr == nil {
		return fmt.Errorf("printer not available")
	}
	if err := a.pr.Print(entry.ArchivePath, entry.Printer); err != nil {
		return fmt.Errorf("ReprintDocument: print error: %w", err)
	}
	return nil
}

// ─── Learning ────────────────────────────────────────────────────────────────

// GetLearningQueue returns the current in-memory learning queue.
func (a *App) GetLearningQueue() []processor.LearningItem {
	a.learningMu.Lock()
	defer a.learningMu.Unlock()
	if len(a.learningQueue) == 0 {
		return []processor.LearningItem{}
	}
	// Return a copy to avoid data races on the slice.
	result := make([]processor.LearningItem, len(a.learningQueue))
	copy(result, a.learningQueue)
	return result
}

// HandleLearning processes a decision for a pending learning item.
func (a *App) HandleLearning(id string, action LearningAction) error {
	// Find and remove the item from the queue.
	a.learningMu.Lock()
	var item *processor.LearningItem
	filtered := a.learningQueue[:0]
	for i := range a.learningQueue {
		if a.learningQueue[i].ID == id {
			cp := a.learningQueue[i]
			item = &cp
		} else {
			filtered = append(filtered, a.learningQueue[i])
		}
	}
	a.learningQueue = filtered
	a.learningMu.Unlock()

	if item == nil {
		return fmt.Errorf("HandleLearning: item %q not found in queue", id)
	}

	switch action.Action {
	case "print", "print_and_memorize":
		// Print the file.
		if a.pr == nil {
			return fmt.Errorf("printer not available")
		}
		a.appLog.Info("Impression manuelle: %s → %s", item.Filename, action.Printer)
		if err := a.pr.Print(item.FilePath, action.Printer); err != nil {
			errMsg := fmt.Sprintf("Échec impression %s sur %s: %v", item.Filename, action.Printer, err)
			runtime.EventsEmit(a.ctx, "print:error", errMsg)
			a.appLog.Error("%s", errMsg)
			a.sendWebhook("print_error", errMsg)
			return fmt.Errorf("HandleLearning: %s", errMsg)
		}
		a.appLog.Info("Imprimé avec succès: %s sur %s", item.Filename, action.Printer)

		// Archive to printed/<archiveSubdir or printer name>.
		subdir := action.ArchiveSubdir
		if subdir == "" {
			subdir = action.Printer
		}
		archPath, err := a.ar.Archive(item.FilePath, "printed", subdir, action.Printer)
		if err != nil {
			a.appLog.Error("Erreur archivage %s: %v", item.Filename, err)
		}

		// Insert history record.
		if a.hist != nil {
			_ = a.hist.Insert(history.HistoryEntry{
				Filename:     item.Filename,
				OriginalPath: item.FilePath,
				ArchivePath:  archPath,
				Printer:      action.Printer,
				RuleName:     action.RuleName,
				Status:       "printed",
				PageCount:    item.PageCount,
				PageWidthMM:  item.PageWidthMM,
				PageHeightMM: item.PageHeightMM,
			})
		}

		// If "print_and_memorize", persist a new rule and reload classifier.
		if action.Action == "print_and_memorize" && action.RuleName != "" {
			filenameRegex := action.FilenameRegex
			if filenameRegex == "" {
				filenameRegex = "(?i)" + item.Filename
			}
			newRule := rules.Rule{
				Name: action.RuleName,
				Match: rules.RuleMatch{
					FilenameRegex:      filenameRegex,
					PageWidthMM:        item.PageWidthMM,
					PageHeightMM:       item.PageHeightMM,
					DimensionTolerance: 5,
				},
				Action: rules.RuleAction{
					Printer:       action.Printer,
					ArchiveSubdir: subdir,
				},
			}
			if err := a.rulesStore.Add(newRule); err != nil {
				a.appLog.Error("Erreur ajout règle '%s': %v", action.RuleName, err)
			} else {
				a.appLog.Info("Nouvelle règle créée: '%s' → %s", action.RuleName, action.Printer)
				if err := a.reloadClassifier(); err != nil {
					a.appLog.Error("Erreur rechargement classifier: %v", err)
				}
			}
		}

	case "ignore":
		a.appLog.Info("Fichier ignoré: %s", item.Filename)
		// Archive to unknown.
		archPath, err := a.ar.Archive(item.FilePath, "unknown", "", "")
		if err != nil {
			a.appLog.Error("Erreur archivage %s: %v", item.Filename, err)
		}
		if a.hist != nil {
			_ = a.hist.Insert(history.HistoryEntry{
				Filename:     item.Filename,
				OriginalPath: item.FilePath,
				ArchivePath:  archPath,
				Status:       "ignored",
				PageCount:    item.PageCount,
				PageWidthMM:  item.PageWidthMM,
				PageHeightMM: item.PageHeightMM,
			})
		}

	default:
		return fmt.Errorf("HandleLearning: unknown action %q", action.Action)
	}

	runtime.EventsEmit(a.ctx, "learning:resolved", id)
	runtime.EventsEmit(a.ctx, "file:processed")
	return nil
}

// GetDocumentBase64 returns the archived PDF content as a base64-encoded string.
func (a *App) GetDocumentBase64(id int64) (string, error) {
	if a.hist == nil {
		return "", fmt.Errorf("history database not available")
	}
	entry, err := a.hist.GetByID(id)
	if err != nil {
		return "", fmt.Errorf("GetDocumentBase64: %w", err)
	}
	if entry == nil {
		return "", fmt.Errorf("document %d not found", id)
	}
	if entry.ArchivePath == "" {
		return "", fmt.Errorf("no archive path for document %d", id)
	}
	data, err := os.ReadFile(entry.ArchivePath)
	if err != nil {
		return "", fmt.Errorf("read archive file: %w", err)
	}
	return base64Encode(data), nil
}

// autoCleanupLoop runs cleanup once at startup then every 24 hours.
func (a *App) autoCleanupLoop() {
	// Run once at startup
	a.runCleanup()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.runCleanup()
		case <-a.cleanupStop:
			return
		}
	}
}

// runCleanup deletes history entries and archive files older than AutoCleanupDays.
func (a *App) runCleanup() {
	days := a.cfg.AutoCleanupDays
	if days <= 0 || a.hist == nil {
		return
	}

	paths, err := a.hist.DeleteOlderThan(days)
	if err != nil {
		log.Printf("[app] auto-cleanup error: %v", err)
		return
	}

	deleted := 0
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[app] cleanup: could not delete %s: %v", p, err)
			}
		} else {
			deleted++
		}
	}

	if len(paths) > 0 {
		log.Printf("[app] auto-cleanup: removed %d history entries, deleted %d archive files (older than %d days)", len(paths), deleted, days)
	}
}

// RunCleanupNow allows the user to trigger a manual cleanup from the UI.
func (a *App) RunCleanupNow() (int, error) {
	days := a.cfg.AutoCleanupDays
	if days <= 0 {
		return 0, fmt.Errorf("auto-cleanup is disabled (set a retention period first)")
	}
	if a.hist == nil {
		return 0, fmt.Errorf("history database not available")
	}

	paths, err := a.hist.DeleteOlderThan(days)
	if err != nil {
		return 0, err
	}

	for _, p := range paths {
		if p != "" {
			os.Remove(p)
		}
	}
	return len(paths), nil
}

// ─── Printer status & test ────────────────────────────────────────────────────

// PrinterStatus holds the online state of a printer.
type PrinterStatus struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

// GetPrinterStatuses returns the online/offline state of all printers.
func (a *App) GetPrinterStatuses() []PrinterStatus {
	printers := a.GetPrinters()
	statuses := make([]PrinterStatus, len(printers))
	for i, name := range printers {
		statuses[i] = PrinterStatus{Name: name, Online: a.pr != nil && a.pr.IsOnline(name)}
	}
	return statuses
}

// GetPrinterStats returns per-printer document/page counts for today.
func (a *App) GetPrinterStatsToday() []history.PrinterStats {
	if a.hist == nil {
		return []history.PrinterStats{}
	}
	stats, err := a.hist.GetPrinterStats()
	if err != nil {
		log.Printf("[app] GetPrinterStats error: %v", err)
		return []history.PrinterStats{}
	}
	return stats
}

// TestPrint sends a test page to the specified printer.
func (a *App) TestPrint(printerName string) error {
	if a.pr == nil {
		return fmt.Errorf("printer not available")
	}
	err := a.pr.TestPrint(printerName)
	if err != nil {
		a.appLog.Error("Test print failed on %s: %v", printerName, err)
	} else {
		a.appLog.Info("Test print sent to %s", printerName)
	}
	return err
}

// ─── Batch reprint ───────────────────────────────────────────────────────────

// BatchReprint reprints multiple documents by their IDs.
func (a *App) BatchReprint(ids []int64) (int, error) {
	if a.hist == nil || a.pr == nil {
		return 0, fmt.Errorf("printer or history not available")
	}
	success := 0
	for _, id := range ids {
		entry, err := a.hist.GetByID(id)
		if err != nil || entry == nil {
			continue
		}
		if err := a.pr.Print(entry.ArchivePath, entry.Printer); err != nil {
			a.appLog.Error("Batch reprint failed for %s: %v", entry.Filename, err)
			continue
		}
		success++
	}
	a.appLog.Info("Batch reprint: %d/%d documents", success, len(ids))
	return success, nil
}

// ─── Statistics ──────────────────────────────────────────────────────────────

// GetDailyStatsChart returns daily print/fail counts for the last N days.
func (a *App) GetDailyStatsChart(days int) []history.DailyStats {
	if a.hist == nil {
		return []history.DailyStats{}
	}
	stats, err := a.hist.GetDailyStats(days)
	if err != nil {
		log.Printf("[app] GetDailyStats error: %v", err)
		return []history.DailyStats{}
	}
	return stats
}

// ─── Rule reorder ────────────────────────────────────────────────────────────

// ReorderRules reorders rules by the given list of names.
func (a *App) ReorderRules(names []string) error {
	if err := a.rulesStore.Reorder(names); err != nil {
		return err
	}
	return a.reloadClassifier()
}

// UpdateRule updates an existing rule by name.
func (a *App) UpdateRule(name string, rule rules.Rule) error {
	if err := a.rulesStore.Update(name, rule); err != nil {
		return err
	}
	return a.reloadClassifier()
}

// ─── Pause / Resume ──────────────────────────────────────────────────────────

// PauseWatcher pauses the file watcher.
func (a *App) PauseWatcher() {
	a.stopWatcher()
	a.paused = true
	a.appLog.Info("Watcher paused")
}

// ResumeWatcher resumes the file watcher.
func (a *App) ResumeWatcher() {
	if a.cfg.WatchDir != "" {
		a.startWatcher(a.cfg.WatchDir)
	}
	// Also start extra watch dirs
	for _, dir := range a.cfg.ExtraWatchDirs {
		if dir != "" {
			a.startExtraWatcher(dir)
		}
	}
	a.paused = false
	a.appLog.Info("Watcher resumed")
}

// IsPaused returns whether the watcher is paused.
func (a *App) IsPaused() bool {
	return a.paused
}

// ─── Multi-folder ────────────────────────────────────────────────────────────

func (a *App) startExtraWatcher(dir string) {
	delay := a.cfg.StabilityDelaySeconds
	if delay <= 0 {
		delay = 2
	}
	w, err := watcher.New(dir, a.fileChan, delay)
	if err != nil {
		log.Printf("[app] extra watcher init error for %s: %v", dir, err)
		return
	}
	go func() {
		if err := w.Start(); err != nil {
			log.Printf("[app] extra watcher stopped for %s: %v", dir, err)
		}
	}()
}

// ─── Logs ────────────────────────────────────────────────────────────────────

// GetLogs returns the in-memory log entries.
func (a *App) GetLogs() []applog.LogEntry {
	if a.appLog == nil {
		return []applog.LogEntry{}
	}
	return a.appLog.GetEntries()
}

// ClearLogs clears the in-memory log buffer.
func (a *App) ClearLogs() {
	if a.appLog != nil {
		a.appLog.Clear()
	}
}

// ─── Import config ───────────────────────────────────────────────────────────

// ImportConfig opens a file dialog and imports a PrintDock config export file.
func (a *App) ImportConfig() error {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Importer une configuration PrintDock",
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML Files", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil {
		return fmt.Errorf("open dialog: %w", err)
	}
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	content := string(data)
	parts := strings.SplitN(content, "# === PrintDock Rules ===", 2)
	if len(parts) == 2 {
		cfgPart := strings.TrimPrefix(parts[0], "# === PrintDock Configuration ===\n")
		rulesPart := strings.TrimSpace(parts[1])
		if err := os.WriteFile(a.configPath, []byte(cfgPart), 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := os.WriteFile(a.rulesPath, []byte(rulesPart), 0644); err != nil {
			return fmt.Errorf("write rules: %w", err)
		}
	} else {
		// Try as config-only
		if err := os.WriteFile(a.configPath, data, 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
	}

	// Reload
	cfg, err := config.Load(a.configPath)
	if err == nil {
		a.cfg = cfg
	}
	if err := a.reloadClassifier(); err != nil {
		log.Printf("[app] import: classifier reload error: %v", err)
	}
	a.appLog.Info("Configuration imported from %s", path)
	return nil
}

// ─── Webhook ─────────────────────────────────────────────────────────────────

// SendWebhook sends a POST request to the configured webhook URL.
func (a *App) sendWebhook(event, message string) {
	url := a.cfg.WebhookURL
	if url == "" {
		return
	}
	go func() {
		body := fmt.Sprintf(`{"event":"%s","message":"%s","timestamp":"%s"}`,
			event, message, time.Now().Format(time.RFC3339))
		resp, err := http.Post(url, "application/json", strings.NewReader(body))
		if err != nil {
			a.appLog.Warn("Webhook error: %v", err)
			return
		}
		resp.Body.Close()
	}()
}

// ─── Analyze PDF ─────────────────────────────────────────────────────────────

// PDFAnalysisResult holds the analysis of a PDF file, including any matching rule.
type PDFAnalysisResult struct {
	Item         processor.LearningItem `json:"item"`
	MatchedRule  *rules.Rule            `json:"matchedRule"`  // nil if no match
}

// AnalyzePDFPath analyzes a PDF at the given path. If it matches a rule, returns the rule.
// Otherwise adds it to the learning queue.
func (a *App) AnalyzePDFPath(path string) (*PDFAnalysisResult, error) {
	return a.analyzePDFPath(path)
}

func (a *App) analyzePDFPath(path string) (*PDFAnalysisResult, error) {
	info, err := processor.ExtractPDFInfo(path)
	if err != nil {
		return nil, fmt.Errorf("analyze PDF: %w", err)
	}
	filename := filepath.Base(path)
	item := processor.LearningItem{
		ID:           fmt.Sprintf("manual-%d", time.Now().UnixNano()),
		FilePath:     path,
		Filename:     filename,
		PageCount:    info.PageCount,
		PageWidthMM:  info.WidthMM,
		PageHeightMM: info.HeightMM,
		DetectedAt:   time.Now(),
	}

	// Check if it matches an existing rule
	matchedRule := a.cl.Classify(filename, info.WidthMM, info.HeightMM, info.PageCount)

	if matchedRule != nil {
		// Return the match info — don't add to learning queue
		return &PDFAnalysisResult{Item: item, MatchedRule: matchedRule}, nil
	}

	// No match — add to learning queue
	a.learningMu.Lock()
	a.learningQueue = append(a.learningQueue, item)
	a.learningMu.Unlock()
	runtime.EventsEmit(a.ctx, "learning:new", item)
	return &PDFAnalysisResult{Item: item, MatchedRule: nil}, nil
}

// AnalyzePDF opens a file dialog version of AnalyzePDFPath.
func (a *App) AnalyzePDF() (*PDFAnalysisResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Sélectionner un fichier PDF",
		Filters: []runtime.FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	return a.analyzePDFPath(path)
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// ─── Directory picker ─────────────────────────────────────────────────────────

// ShowWindow shows the main window and brings it to focus.
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// PickDirectory opens a native directory picker dialog and returns the selected path.
func (a *App) PickDirectory(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

// ─── Export / Reset ──────────────────────────────────────────────────────────

// ExportConfig returns the content of config.yaml and rules.yaml as a combined YAML string.
func (a *App) ExportConfig() (string, error) {
	cfgData, err := os.ReadFile(a.configPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read config: %w", err)
	}
	rulesData, err := os.ReadFile(a.rulesPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read rules: %w", err)
	}
	result := "# === PrintDock Configuration ===\n"
	result += string(cfgData)
	result += "\n# === PrintDock Rules ===\n"
	result += string(rulesData)
	return result, nil
}

// SaveExportFile opens a native save dialog and writes the exported config to the chosen path.
func (a *App) SaveExportFile() error {
	content, err := a.ExportConfig()
	if err != nil {
		return err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exporter la configuration PrintDock",
		DefaultFilename: "printdock-export.yaml",
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML Files", Pattern: "*.yaml;*.yml"},
		},
	})
	if err != nil {
		return fmt.Errorf("save dialog: %w", err)
	}
	if path == "" {
		return nil // user cancelled
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ResetHistory drops all rows from the history table.
func (a *App) ResetHistory() error {
	if a.hist == nil {
		return fmt.Errorf("history database not available")
	}
	return a.hist.Reset()
}

// IsSumatraInstalled checks if SumatraPDF is available for printing.
func (a *App) IsSumatraInstalled() bool {
	if a.pr == nil {
		return false
	}
	return a.pr.IsSumatraInstalled()
}

// DownloadSumatra downloads SumatraPDF to the application directory.
func (a *App) DownloadSumatra() error {
	if a.pr == nil {
		return fmt.Errorf("printer module not available")
	}
	a.appLog.Info("Downloading SumatraPDF...")
	if err := a.pr.DownloadSumatra(); err != nil {
		a.appLog.Error("SumatraPDF download failed: %v", err)
		return err
	}
	a.appLog.Info("SumatraPDF downloaded successfully")
	return nil
}
