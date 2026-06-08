package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultExcelPath              = "bonds.xlsx"
	defaultInputSheetName         = "Sheet1"
	defaultOutputSheetName        = "Sheet2"
	defaultBrokerSellFeePercent   = 0.3
	defaultExchangeSellFeePercent = 0.085
	defaultTaxRatePercent         = 13
)

type Config struct {
	ExcelPath              string  `yaml:"excel_path"`
	InputSheetName         string  `yaml:"input_sheet_name"`
	OutputSheetName        string  `yaml:"output_sheet_name"`
	BrokerSellFeePercent   float64 `yaml:"broker_sell_fee_percent"`
	ExchangeSellFeePercent float64 `yaml:"exchange_sell_fee_percent"`
	TaxRatePercent         float64 `yaml:"tax_rate_percent"`
}

func getDefaultConfig() Config {
	return Config{
		ExcelPath:              defaultExcelPath,
		InputSheetName:         defaultInputSheetName,
		OutputSheetName:        defaultOutputSheetName,
		BrokerSellFeePercent:   defaultBrokerSellFeePercent,
		ExchangeSellFeePercent: defaultExchangeSellFeePercent,
		TaxRatePercent:         defaultTaxRatePercent,
	}
}

func Load(fileName string) (*Config, error) {
	cfg := getDefaultConfig()
	// Return default config if config file name is not defined
	if fileName == "" {
		return &cfg, nil
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}

	return &cfg, nil
}

func MustLoad(fileName string) *Config {
	cfg, err := Load(fileName)
	if err != nil {
		panic(err)
	}

	return cfg
}
