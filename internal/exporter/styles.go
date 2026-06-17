package exporter

import "github.com/xuri/excelize/v2"

// headerStyle returns the style for header rows.
func headerStyle(f *excelize.File) (int, error) {
	style := &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#FFFFFF",
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#4472C4"},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#C0C0C0", Style: 1},
		},
	}
	return f.NewStyle(style)
}

// dataStyle returns the style for data rows.
func dataStyle(f *excelize.File) (int, error) {
	style := &excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Family: "Arial",
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
	}
	return f.NewStyle(style)
}

// setColumnWidths sets reasonable column widths for each sheet.
func setColumnWidths(f *excelize.File, sheet string, cols []string) {
	for i, col := range cols {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		width := float64(len(col)) * 1.5
		if width < 12 {
			width = 12
		}
		if width > 60 {
			width = 60
		}
		f.SetColWidth(sheet, colName, colName, width)
	}
}
