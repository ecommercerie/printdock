package processor

import (
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// PDFInfo holds basic metadata extracted from a PDF file.
type PDFInfo struct {
	PageCount int
	WidthMM   float64
	HeightMM  float64
}

// pointsToMM converts PDF points to millimetres (1 point = 25.4/72 mm).
const pointsToMM = 25.4 / 72.0

// ExtractPDFInfo reads a PDF file and returns its page count and the
// dimensions of the first page in millimetres.
func ExtractPDFInfo(path string) (*PDFInfo, error) {
	ctx, err := api.ReadContextFile(path)
	if err != nil {
		return nil, err
	}
	if err := api.OptimizeContext(ctx); err != nil {
		return nil, err
	}

	info := &PDFInfo{
		PageCount: ctx.XRefTable.PageCount,
	}

	if ctx.XRefTable.PageCount > 0 {
		dims, err := ctx.XRefTable.PageDims()
		if err != nil {
			return nil, err
		}
		if len(dims) > 0 {
			info.WidthMM = dims[0].Width * pointsToMM
			info.HeightMM = dims[0].Height * pointsToMM
		}
	}

	return info, nil
}
