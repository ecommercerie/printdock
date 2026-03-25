package processor

import (
	"path/filepath"
	"testing"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// createTestPDF writes a single-page PDF with the given dimensions (in points) to path.
func createTestPDF(path string, widthPt, heightPt float64) error {
	xref, err := pdfcpu.CreateXRefTableWithRootDict()
	if err != nil {
		return err
	}

	mediaBox := types.RectForDim(widthPt, heightPt)

	pagesDict := types.Dict{
		"Type":  types.Name("Pages"),
		"Kids":  types.Array{},
		"Count": types.Integer(0),
	}
	pagesIndRef, err := xref.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	pageDict := types.Dict{
		"Type":      types.Name("Page"),
		"MediaBox":  mediaBox.Array(),
		"Parent":    *pagesIndRef,
		"Resources": types.Dict{},
	}
	pageIndRef, err := xref.IndRefForNewObject(pageDict)
	if err != nil {
		return err
	}

	entry, _ := xref.FindTableEntryLight(pagesIndRef.ObjectNumber.Value())
	if pd, ok := entry.Object.(types.Dict); ok {
		pd["Kids"] = types.Array{*pageIndRef}
		pd["Count"] = types.Integer(1)
	}
	xref.PageCount = 1

	catalog, err := xref.Catalog()
	if err != nil {
		return err
	}
	catalog["Pages"] = *pagesIndRef

	return pdfcpuapi.CreatePDFFile(xref, path, model.NewDefaultConfiguration())
}

func TestExtractPDFInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")

	// A4 in points: 595.28 x 841.89
	if err := createTestPDF(path, 595.28, 841.89); err != nil {
		t.Fatalf("createTestPDF: %v", err)
	}

	info, err := ExtractPDFInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.PageCount != 1 {
		t.Errorf("expected 1 page, got %d", info.PageCount)
	}
	// A4 = 210 x 297 mm (from 595.28 x 841.89 points)
	if info.WidthMM < 200 || info.WidthMM > 220 {
		t.Errorf("expected width ~210mm, got %.1f", info.WidthMM)
	}
	if info.HeightMM < 290 || info.HeightMM > 305 {
		t.Errorf("expected height ~297mm, got %.1f", info.HeightMM)
	}
}
