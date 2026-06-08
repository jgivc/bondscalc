package domain

import (
	"fmt"
	"math"
	"slices"
	"time"
)

const (
	xirrDataInitialArrayLength = 3
)

type Lot struct {
	Date     time.Time
	SECID    string
	Quantity int
	// Цена всех бумаг
	BuyPrice float64
	// Уплаченный пр покупке НКД всех бумаг
	BuyNKD float64
	// Общая комиссия брокера
	BuyBrokerFee float64
	// Общая комиссия биржи
	BuyExchangeFee float64
}

func (l *Lot) PurchaseCosts() float64 {
	return l.BuyPrice + l.BuyNKD + l.BuyBrokerFee + l.BuyExchangeFee
}

type CouponPayment struct {
	Date  time.Time
	Value float64
}

type MarketInfo struct {
	SECID        string
	Name         string
	LotValue     float64
	CurrentPrice float64
	CurrentNKD   float64
}

type ConfigRates struct {
	BrokerSellFeePercent   float64
	ExchangeSellFeePercent float64
	TaxRatePercent         float64
}

type CouponData struct {
	Date  time.Time
	Value float64
}

type CouponsData []CouponData

func (d CouponsData) Total() float64 {
	var total float64

	for _, cd := range d {
		total += cd.Value
	}

	return total
}

type Position struct {
	SECID      string
	MarketInfo MarketInfo
	Lots       []Lot
	Coupons    []CouponPayment
}

func (p *Position) normalizeDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func (p *Position) CouponsReceived(lot Lot) CouponsData {
	quantity := float64(lot.Quantity)
	totalCoupons := make(CouponsData, 0)
	start := p.normalizeDate(lot.Date)
	stop := p.normalizeDate(time.Now())

	for _, coupon := range p.Coupons {
		couponDate := p.normalizeDate(coupon.Date)
		if couponDate.Before(start) {
			continue
		}

		if couponDate.After(stop) {
			break
		}

		totalCoupons = append(totalCoupons, CouponData{
			Date:  couponDate,
			Value: coupon.Value * quantity,
		})
	}

	return totalCoupons
}

func (p *Position) Calculate(cfg ConfigRates) *CalculationResult {
	res := &CalculationResult{
		SECID:    p.SECID,
		Name:     p.MarketInfo.Name,
		XirrData: make([]XirrDataRecord, 0, xirrDataInitialArrayLength),
	}

	slices.SortFunc(p.Coupons, func(a, b CouponPayment) int {
		return a.Date.Compare(b.Date)
	})

	dateNow := p.normalizeDate(time.Now())

	var totalCoupons float64
	for _, lot := range p.Lots {
		res.Quantity += lot.Quantity
		res.Spent += lot.PurchaseCosts()
		res.AddXirrSpent(p.normalizeDate(lot.Date), lot.PurchaseCosts(), "Покупка %d шт., включая НКД %2.f", lot.Quantity, lot.BuyNKD)

		couponsReceived := p.CouponsReceived(lot)
		for _, coupons := range couponsReceived {
			if cfg.TaxRatePercent > 0 {
				couponTaxes := math.Round(coupons.Value * (cfg.TaxRatePercent / 100))
				res.AddXirrRevenue(coupons.Date, coupons.Value-couponTaxes, "Выплата купонов, без налога %.2f%%", cfg.TaxRatePercent)
				res.Taxes += couponTaxes
			} else {
				res.AddXirrRevenue(coupons.Date, coupons.Value, "Выплата купонов")
			}
		}
		totalCoupons += couponsReceived.Total()
	}

	slices.SortFunc(res.XirrData, func(a, b XirrDataRecord) int {
		return a.Date.Compare(b.Date)
	})

	// текущая стоимость + НКД
	res.Revenue = (p.MarketInfo.CurrentPrice / 100) * p.MarketInfo.LotValue * float64(res.Quantity)
	res.Revenue += p.MarketInfo.CurrentNKD * float64(res.Quantity)

	// Комиссия
	res.Revenue -= res.Revenue/100*cfg.BrokerSellFeePercent + res.Revenue/100*cfg.ExchangeSellFeePercent
	res.AddXirrRevenue(dateNow, res.Revenue, "Продажа бумаг + НКД, %d шт.", res.Quantity)

	// Налоги
	if cfg.TaxRatePercent > 0 {
		if sum := res.Revenue - res.Spent; sum > 0 {
			res.Taxes += math.Round(sum * (cfg.TaxRatePercent / 100))
			res.AddXirrSpent(dateNow, res.Taxes, "Налоги %.2f%% с курсовой разницы", cfg.TaxRatePercent)
		}
	}

	// Полученные купоны
	if totalCoupons > 0 {
		res.Revenue += totalCoupons
	}

	res.NetProfit = res.Revenue - res.Spent - res.Taxes
	res.YieldPercent = res.NetProfit / res.Spent * 100

	return res
}

type XirrDataRecord struct {
	Date    time.Time
	Value   float64
	Comment string
}

type CalculationResult struct {
	SECID    string
	Name     string
	Quantity int
	// Затраты
	Spent float64
	// Прибыль
	Revenue float64
	// Налоги
	Taxes float64
	// Чистая прибыль
	NetProfit float64
	// Процент доходности
	YieldPercent float64
	// Данные для ЧИСТВНДОХ (XIRR)
	XirrData []XirrDataRecord
}

func (r *CalculationResult) AddXirrRevenue(date time.Time, val float64, format string, a ...any) {
	r.XirrData = append(r.XirrData, XirrDataRecord{
		Date:    date,
		Value:   val,
		Comment: fmt.Sprintf(format, a...),
	})
}

func (r *CalculationResult) AddXirrSpent(date time.Time, val float64, format string, a ...any) {
	r.AddXirrRevenue(date, val*-1, format, a...)
}
