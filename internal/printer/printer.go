package printer

// SumatraStatus holds information about the SumatraPDF installation.
type SumatraStatus struct {
	Installed      bool   `json:"installed"`
	CurrentVersion string `json:"currentVersion,omitempty"` // e.g. "3.5.2"
	LatestVersion  string `json:"latestVersion,omitempty"`  // from GitHub
	UpdateAvail    bool   `json:"updateAvailable"`
}

type Printer interface {
	Print(filePath, printerName string) error
	ListPrinters() ([]string, error)
	IsOnline(printerName string) bool
	TestPrint(printerName string) error
	IsSumatraInstalled() bool
	DownloadSumatra() error
	GetSumatraStatus() SumatraStatus
	Close() error
}
