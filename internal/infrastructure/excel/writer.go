package excel

import (
	"fmt"
	"log/slog"

	"github.com/jgivc/bondscalc/internal/domain"
	"github.com/xuri/excelize/v2"
)

const (
	StyleDate int = iota
	StyleCurrency
	StyleHeader
	StylePercent
	StyleDefault
	StyleSecName

	formatDate     = "yyyy-mm-dd"
	formatCurrency = "#,##0.00 [$р.-419];-#,##0.00 [$р.-419]"
	formatPercent  = "0.00%"
)

type ExcelWriter struct {
	sheetPath string
	sheetName string
	log       *slog.Logger
}

func NewExcelWriter(sheetPath string, sheetName string, log *slog.Logger) *ExcelWriter {
	return &ExcelWriter{
		sheetPath: sheetPath,
		sheetName: sheetName,
		log:       log.With(slog.String("struct", "ExcelWriter")),
	}
}

// type ColumnDef struct {
// 	Header  string                                       // Название в шапке (A1)
// 	Width   float64                                      // Ширина колонки
// 	StyleID int                                          // ID стиля (проценты, рубли)
// 	Extract func(r domain.CalculationResult) interface{} // Функция извлечения данных
// }

type ColumnDefXirr struct {
	Width   float64
	StyleID int
	Value   func(d domain.XirrDataRecord) any
}

func (r *ExcelWriter) initStyles(f *excelize.File) (map[int]int, error) {
	styles := map[int]int{}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4F81BD"}, Pattern: 1},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create header style: %w", err)
	}
	styles[StyleHeader] = headerStyle

	fmtCur := formatCurrency
	currStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &fmtCur})
	if err != nil {
		return nil, fmt.Errorf("cannot create currency style: %w", err)
	}
	styles[StyleCurrency] = currStyle

	fmtPercent := formatPercent
	pctStyle, err := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: &fmtPercent,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create percent style: %w", err)
	}
	styles[StylePercent] = pctStyle

	fmtDate := formatDate
	dateStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &fmtDate,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create date style: %w", err)
	}
	styles[StyleDate] = dateStyle

	defStyle, err := f.NewStyle(&excelize.Style{})
	if err != nil {
		return nil, fmt.Errorf("cannot create default style: %w", err)
	}
	styles[StyleDefault] = defStyle

	secNameStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create sec name style: %w", err)
	}
	styles[StyleSecName] = secNameStyle

	return styles, nil
}

func (r *ExcelWriter) WriteResults(results []domain.CalculationResult) error {
	log := r.log.With(slog.String("op", "WriteResult"))

	f, err := excelize.OpenFile(r.sheetPath)
	if err != nil {
		log.Error("Cannot open excel file", slog.String("path", r.sheetPath), slog.Any("error", err))

		return fmt.Errorf("cannot open excel file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Cannot close excel file", slog.String("path", r.sheetPath), slog.Any("error", err))
		}
	}()

	sheetIndex, err := f.GetSheetIndex(r.sheetName)
	if err != nil {
		log.Error("Cannot get sheet index", slog.String("sheet", r.sheetName), slog.Any("error", err))

		return fmt.Errorf("cannot get sheet name: %w", err)
	}
	if sheetIndex > -1 {
		if err := f.DeleteSheet(r.sheetName); err != nil {
			log.Error("Cannot remove sheet", slog.String("sheet", r.sheetName), slog.Any("error", err))

			return fmt.Errorf("cannot remove sheet: %w", err)
		}
	}

	if _, err := f.NewSheet(r.sheetName); err != nil {
		log.Error("Cannot create sheet", slog.String("sheet", r.sheetName), slog.Any("error", err))

		return fmt.Errorf("cannot create sheet: %w", err)
	}

	styles, err := r.initStyles(f)
	if err != nil {
		log.Error("Cannot initialize styles", slog.String("sheet", r.sheetName), slog.Any("error", err))

		return fmt.Errorf("cannot initialize styles: %w", err)
	}

	colls := []ColumnDefXirr{
		{
			StyleID: styles[StyleDate],
			Width:   15,
			Value: func(d domain.XirrDataRecord) any {
				return d.Date
			},
		},
		{
			StyleID: styles[StyleCurrency],
			Width:   20,
			Value: func(d domain.XirrDataRecord) any {
				return d.Value
			},
		},
		{
			StyleID: styles[StyleDefault],
			Width:   50,
			Value: func(d domain.XirrDataRecord) any {
				return d.Comment
			},
		},
	}

	nameLength := 0
	row, col := 1, 1
	for _, result := range results {
		cellAxis, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			return fmt.Errorf("cannot translate coordinate to cell name: %w", err)
		}

		name := fmt.Sprintf("%s (%s)", result.Name, result.SECID)
		if len(name) > nameLength {
			nameLength = len(name)
		}
		if err := f.SetCellStr(r.sheetName, cellAxis, name); err != nil {
			return fmt.Errorf("cannot set SECID value to cell: %w", err)
		}

		// cellAxis, _ = excelize.CoordinatesToCellName(col+1, row)
		// if err := f.SetCellStr(r.sheetName, cellAxis, result.Name); err != nil {
		// 	return fmt.Errorf("cannot set SECID value to cell: %w", err)
		// }
		// row++

		xirrStartRow := row
		for _, data := range result.XirrData {
			for i, c := 0, col+1; i < 3; i, c = i+1, c+1 {
				cellAxis, _ := excelize.CoordinatesToCellName(c, row)
				f.SetCellValue(r.sheetName, cellAxis, colls[i].Value(data))
				f.SetCellStyle(r.sheetName, cellAxis, cellAxis, colls[i].StyleID)
			}
			row++
		}

		for i, c := 0, col+1; i < 3; i, c = i+1, c+1 {
			colName, _ := excelize.ColumnNumberToName(c)
			if width := colls[i].Width; width > 0 {
				if err := f.SetColWidth(r.sheetName, colName, colName, width); err != nil {
					return fmt.Errorf("cannot set column width: %w", err)
				}
			}
		}

		valuesStart, _ := excelize.CoordinatesToCellName(col+2, xirrStartRow)
		valuesEnd, _ := excelize.CoordinatesToCellName(col+2, row-1)

		datesStart, _ := excelize.CoordinatesToCellName(col+1, xirrStartRow)
		datesEnd, _ := excelize.CoordinatesToCellName(col+1, row-1)

		cellAxis, _ = excelize.CoordinatesToCellName(col+2, row)
		f.SetCellFormula(r.sheetName, cellAxis, fmt.Sprintf("XIRR(%s:%s,%s:%s)/100", valuesStart, valuesEnd, datesStart, datesEnd))
		f.SetCellStyle(r.sheetName, cellAxis, cellAxis, styles[StylePercent])

		mergeRowStart, _ := excelize.CoordinatesToCellName(col, xirrStartRow)
		mergeRowEnd, _ := excelize.CoordinatesToCellName(col, row)
		f.MergeCell(r.sheetName, mergeRowStart, mergeRowEnd)
		f.SetCellStyle(r.sheetName, mergeRowStart, mergeRowEnd, styles[StyleSecName])

		row++
	}

	colName, _ := excelize.ColumnNumberToName(1)
	if err := f.SetColWidth(r.sheetName, colName, colName, float64(nameLength+3)); err != nil {
		return fmt.Errorf("cannot set column width: %w", err)
	}

	if err := f.Save(); err != nil {
		log.Error("Cannot save excel file", slog.Any("error", err))

		return fmt.Errorf("cannot save excel file: %w", err)
	}

	return nil
}
