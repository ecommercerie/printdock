package printer

type Printer interface {
	Print(filePath, printerName string) error
	ListPrinters() ([]string, error)
	IsOnline(printerName string) bool
	TestPrint(printerName string) error
	IsSumatraInstalled() bool
	DownloadSumatra() error
	Close() error
}
