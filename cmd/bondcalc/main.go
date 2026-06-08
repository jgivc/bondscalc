package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/jgivc/bondscalc/internal/infrastructure/config"
	"github.com/jgivc/bondscalc/internal/infrastructure/excel"
	"github.com/jgivc/bondscalc/internal/infrastructure/moex"
	"github.com/jgivc/bondscalc/internal/usecase"
)

func main() {
	cfgPath := flag.String("c", "", "Path to config file")

	flag.Parse()

	// lo := &slog.HandlerOptions{}
	// switch a.cfg.LogLevel {
	// case config.LogLevelInfo:
	// 	lo.Level = slog.LevelInfo
	// case config.LogLevelWarn:
	// 	lo.Level = slog.LevelWarn
	// case config.LogLevelError:
	// 	lo.Level = slog.LevelError
	// case config.LogLevelDebug:
	// 	lo.Level = slog.LevelDebug
	// default:
	// 	panic("unknown log level")
	// }
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.MustLoad(*cfgPath)

	portfolioReader := excel.NewExcelReader(cfg.ExcelPath, cfg.InputSheetName, log)
	portfolioWriter := excel.NewExcelWriter(cfg.ExcelPath, cfg.OutputSheetName, log)
	marketDataProvider := moex.NewMoexAPI(log)

	calculatorUC := usecase.NewCalculatorUseCase(cfg, portfolioReader, portfolioWriter, marketDataProvider, log)
	calculatorUC.Run()
}
