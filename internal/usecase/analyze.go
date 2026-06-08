package usecase

import (
	"log/slog"

	"github.com/jgivc/bondscalc/internal/domain"
	"github.com/jgivc/bondscalc/internal/infrastructure/config"
)

type PortfolioReader interface {
	ReadLots() ([]domain.Lot, error)
}

type PortfolioWriter interface {
	WriteResults(results []domain.CalculationResult) error
}

type MarketDataProvider interface {
	GetMarketInfo(secID string) (*domain.MarketInfo, error)
	GetCouponSchedule(secID string) ([]domain.CouponPayment, error)
}

type CalculatorUseCase struct {
	cfg                *config.Config
	portfolioReader    PortfolioReader
	portfolioWriter    PortfolioWriter
	marketDataProvider MarketDataProvider
	log                *slog.Logger
}

func NewCalculatorUseCase(
	cfg *config.Config,
	portfolioReader PortfolioReader,
	portfolioWriter PortfolioWriter,
	marketDataProvider MarketDataProvider,
	log *slog.Logger,
) *CalculatorUseCase {
	return &CalculatorUseCase{
		cfg:                cfg,
		portfolioReader:    portfolioReader,
		portfolioWriter:    portfolioWriter,
		marketDataProvider: marketDataProvider,
		log:                log.With(slog.String("struct", "CalculatorUseCase")),
	}
}

func (c *CalculatorUseCase) Run() error {
	log := slog.With(slog.String("op", "Run"))

	log.Info("Start calculation")

	log.Info("Read lots")
	lots, err := c.portfolioReader.ReadLots()
	if err != nil {
		log.Error("Cannot read lots", slog.Any("error", err))

		return err
	}

	positions := map[string]*domain.Position{}
	for _, lot := range lots {
		secid := lot.SECID

		if _, exists := positions[secid]; !exists {
			position := domain.Position{
				SECID: secid,
			}

			log.Info("Cet moex data", slog.String("SECID", secid))
			marketInfo, err := c.marketDataProvider.GetMarketInfo(secid)
			if err != nil {
				log.Error("Cannot get moex info", slog.String("SECID", secid), slog.Any("error", err))

				continue
			}
			position.MarketInfo = *marketInfo

			log.Info("Cet coupons data", slog.String("SECID", secid))
			couponPayment, err := c.marketDataProvider.GetCouponSchedule(secid)
			if err != nil {
				log.Error("Cannot get coupon info", slog.String("SECID", secid), slog.Any("error", err))

				continue
			}
			position.Coupons = couponPayment

			positions[secid] = &position

		}

		positions[lot.SECID].Lots = append(positions[lot.SECID].Lots, lot)
	}

	if len(positions) < 1 {
		log.Error("No lots found")

		return nil
	}

	cfg := domain.ConfigRates{
		BrokerSellFeePercent:   c.cfg.BrokerSellFeePercent,
		ExchangeSellFeePercent: c.cfg.ExchangeSellFeePercent,
		TaxRatePercent:         c.cfg.TaxRatePercent,
	}
	results := make([]domain.CalculationResult, 0, len(positions))
	for secid, position := range positions {
		log.Info("Calculate", slog.String("SECID", secid))
		result := position.Calculate(cfg)
		results = append(results, *result)
	}

	log.Info("Write results")
	if err := c.portfolioWriter.WriteResults(results); err != nil {
		log.Error("Cannot write resilts", slog.Any("error", err))

		return err
	}

	return nil
}
