package excel

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/jgivc/bondscalc/internal/domain"
	"github.com/xuri/excelize/v2"
)

const (
	byDateFormat = "02.01.2006"
)

var (
	regexDate = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)
)

type ExcelReader struct {
	sheetPath string
	sheetName string
	log       *slog.Logger
}

func NewExcelReader(sheetPath string, sheetName string, log *slog.Logger) *ExcelReader {
	return &ExcelReader{
		sheetPath: sheetPath,
		sheetName: sheetName,
		log:       log.With(slog.String("struct", "ExcelReader")),
	}
}

func (r *ExcelReader) ReadLots() ([]domain.Lot, error) {
	log := r.log.With(slog.String("op", "ReadLots"))

	f, err := excelize.OpenFile(r.sheetPath)
	if err != nil {
		log.Error("Cannot open excel file", slog.String("path", r.sheetPath), slog.Any("error", err))

		return nil, fmt.Errorf("cannot open excel file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Cannot close excel file", slog.String("path", r.sheetPath), slog.Any("error", err))
		}
	}()

	rows, err := f.GetRows(r.sheetName)
	if err != nil {
		log.Error("Cannot get sheet rows", slog.String("path", r.sheetPath), slog.Any("error", err))

		return nil, fmt.Errorf("cannot get sheet rows: %w", err)
	}

	lots := make([]domain.Lot, 0, 5)
	for i, row := range rows {
		if len(row) < 6 {
			continue
		}

		// Date, Name, SECID, Quantity, BuyPrice, BuyNKD, BuyBrokerFee, BuyExchangeFee
		if !regexDate.MatchString(row[0]) {
			log.Info("Skip row because date not found in sirst cell", slog.Int("row", i))

			continue
		}
		date, err := time.Parse(byDateFormat, row[0])
		if err != nil {
			log.Error("Cannot parse date", slog.String("value", row[0]), slog.Any("eror", err))

			continue
		}

		data, err := parseFloats(row, 3, 5)
		if err != nil {
			log.Error("Cannot load data", slog.Any("eror", err))

			continue
		}

		lot := domain.Lot{
			Date:           date,
			SECID:          row[2],
			Quantity:       int(data[0]),
			BuyPrice:       data[1],
			BuyNKD:         data[2],
			BuyBrokerFee:   data[3],
			BuyExchangeFee: data[4],
		}

		lots = append(lots, lot)
	}

	return lots, nil
}

func parseFloats(row []string, startIndex int, count int) ([]float64, error) {
	endIndex := startIndex + count
	if len(row) < endIndex {
		return nil, fmt.Errorf("row index out of range")
	}
	out := make([]float64, count)
	for i, j := 0, startIndex; j < endIndex; i, j = i+1, j+1 {
		f, err := strconv.ParseFloat(row[j], 32)
		if err != nil {
			return nil, fmt.Errorf("cannot parse row data: %w", err)
		}
		out[i] = f
	}

	return out, nil
}
