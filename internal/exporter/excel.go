package exporter

import (
	"fmt"
	"strings"

	"GetRoute/internal/model"
	"github.com/xuri/excelize/v2"
)

const outputFilename = "GetRoute.xlsx"

// Export writes all analysis data to an Excel file.
func Export(
	routes []model.RouteInfo,
	classes []model.ClassInfo,
	frameworks []model.FrameworkInfo,
	components []model.ComponentInfo,
	outputPath string,
) error {
	if outputPath == "" {
		outputPath = outputFilename
	}

	f := excelize.NewFile()
	defer f.Close()

	headerSty, _ := headerStyle(f)

	writeRoutesSheet(f, routes, headerSty)
	writeClassesSheet(f, classes, headerSty)
	writeFrameworkSheet(f, frameworks, headerSty)
	writeComponentsSheet(f, components, headerSty)

	// Remove default sheet if Routes was created.
	if idx, err := f.GetSheetIndex("Routes"); err == nil && idx >= 0 {
		f.DeleteSheet("Sheet1")
		f.SetActiveSheet(0)
	}

	return f.SaveAs(outputPath)
}

func writeRoutesSheet(f *excelize.File, routes []model.RouteInfo, headerSty int) {
	sheet := "Routes"
	f.NewSheet(sheet)
	sheetIdx, _ := f.GetSheetIndex(sheet)
	f.SetActiveSheet(sheetIdx)

	headers := []string{"URL", "HTTP_METHOD", "FRAMEWORK", "CLASS_NAME", "CLASS_PATH", "METHOD_NAME", "SOURCE_JAR"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerSty)
	}

	for i, r := range routes {
		row := i + 2
		methods := strings.Join(r.HTTPMethods, ", ")
		f.SetCellValue(sheet, cellName(1, row), r.URL)
		f.SetCellValue(sheet, cellName(2, row), methods)
		f.SetCellValue(sheet, cellName(3, row), r.Framework)
		f.SetCellValue(sheet, cellName(4, row), r.ClassName)
		f.SetCellValue(sheet, cellName(5, row), r.SourceFile)
		f.SetCellValue(sheet, cellName(6, row), r.MethodName)
		f.SetCellValue(sheet, cellName(7, row), r.ArchiveName)
	}

	setColumnWidths(f, sheet, headers)
	autoFilterRange := fmt.Sprintf("A1:%s%d", colLetter(len(headers)), len(routes)+1)
	f.AutoFilter(sheet, autoFilterRange, nil)
}

func writeClassesSheet(f *excelize.File, classes []model.ClassInfo, headerSty int) {
	sheet := "Classes"
	f.NewSheet(sheet)

	headers := []string{"CLASS_NAME", "CLASS_PATH", "SOURCE_JAR"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerSty)
	}

	for i, c := range classes {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), c.FullName)
		f.SetCellValue(sheet, cellName(2, row), strings.ReplaceAll(c.FullName, ".", "/")+".class")
		f.SetCellValue(sheet, cellName(3, row), c.ArchiveName)
	}

	setColumnWidths(f, sheet, headers)
	autoFilterRange := fmt.Sprintf("A1:%s%d", colLetter(len(headers)), len(classes)+1)
	f.AutoFilter(sheet, autoFilterRange, nil)
}

func writeFrameworkSheet(f *excelize.File, frameworks []model.FrameworkInfo, headerSty int) {
	sheet := "Framework"
	f.NewSheet(sheet)

	headers := []string{"FRAMEWORK", "VERSION", "CONFIDENCE", "EVIDENCE"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerSty)
	}

	for i, fw := range frameworks {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), fw.Name)
		f.SetCellValue(sheet, cellName(2, row), fw.Version)
		f.SetCellValue(sheet, cellName(3, row), fw.Confidence)
		f.SetCellValue(sheet, cellName(4, row), strings.Join(fw.Evidence, "; "))
	}

	setColumnWidths(f, sheet, headers)
	autoFilterRange := fmt.Sprintf("A1:%s%d", colLetter(len(headers)), len(frameworks)+1)
	f.AutoFilter(sheet, autoFilterRange, nil)
}

func writeComponentsSheet(f *excelize.File, components []model.ComponentInfo, headerSty int) {
	sheet := "Components"
	f.NewSheet(sheet)

	headers := []string{"COMPONENT", "TYPE", "VERSION", "SOURCE"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerSty)
	}

	for i, c := range components {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), c.Name)
		f.SetCellValue(sheet, cellName(2, row), c.Type)
		f.SetCellValue(sheet, cellName(3, row), c.Version)
		f.SetCellValue(sheet, cellName(4, row), c.Source)
	}

	setColumnWidths(f, sheet, headers)
	autoFilterRange := fmt.Sprintf("A1:%s%d", colLetter(len(headers)), len(components)+1)
	f.AutoFilter(sheet, autoFilterRange, nil)
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func colLetter(n int) string {
	letter, _ := excelize.ColumnNumberToName(n)
	return letter
}
