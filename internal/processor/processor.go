package processor

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"printdock/internal/applog"
	"printdock/internal/archiver"
	"printdock/internal/classifier"
	"printdock/internal/history"
	"printdock/internal/printer"
)

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
	printer      printer.Printer
	archiver     *archiver.Archiver
	history      *history.DB
	appLog       *applog.Logger
	workerCount  int
	stop         chan struct{}
	wg           sync.WaitGroup
	inFlight     sync.Map
}

func New(fileChan chan string, learningChan chan LearningItem, cl *classifier.Classifier, pr printer.Printer, ar *archiver.Archiver, h *history.DB, workers int, appLog ...*applog.Logger) *Processor {
	p := &Processor{
		fileChan:     fileChan,
		learningChan: learningChan,
		classifier:   cl,
		printer:      pr,
		archiver:     ar,
		history:      h,
		workerCount:  workers,
		stop:         make(chan struct{}),
	}
	if len(appLog) > 0 && appLog[0] != nil {
		p.appLog = appLog[0]
	}
	return p
}

func (p *Processor) logInfo(format string, args ...interface{}) {
	log.Printf(format, args...)
	if p.appLog != nil {
		p.appLog.Info(format, args...)
	}
}

func (p *Processor) logError(format string, args ...interface{}) {
	log.Printf(format, args...)
	if p.appLog != nil {
		p.appLog.Error(format, args...)
	}
}

func (p *Processor) logWarn(format string, args ...interface{}) {
	log.Printf(format, args...)
	if p.appLog != nil {
		p.appLog.Warn(format, args...)
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
	p.logInfo("Nouveau fichier détecté: %s", filename)

	info, err := ExtractPDFInfo(path)
	if err != nil {
		p.logError("Erreur lecture PDF %s: %v", filename, err)
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

	p.logInfo("PDF analysé: %s — %.0fx%.0f mm, %d page(s)", filename, info.WidthMM, info.HeightMM, info.PageCount)

	rule := p.classifier.Classify(filename, info.WidthMM, info.HeightMM, info.PageCount)
	if rule == nil {
		p.logWarn("Aucune règle trouvée pour %s — en attente d'apprentissage", filename)
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

	p.logInfo("Règle '%s' → impression sur %s", rule.Name, rule.Action.Printer)

	if err := p.printer.Print(path, rule.Action.Printer); err != nil {
		p.logError("Échec impression %s sur %s: %v", filename, rule.Action.Printer, err)
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

	p.logInfo("Imprimé avec succès: %s sur %s", filename, rule.Action.Printer)

	archPath, err := p.archiver.Archive(path, "printed", rule.Action.ArchiveSubdir, rule.Action.Printer)
	if err != nil {
		p.logError("Erreur archivage %s: %v", filename, err)
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
