package moex

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jgivc/bondscalc/internal/domain"
)

const (
	securitiesBasePath = "https://iss.moex.com/iss/engines/stock/markets/bonds/boards/TQOB/securities.json"
	couponsURLtemplate = "https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/%s.json"

	paramMeta          = "iss.meta"
	paramOnly          = "iss.only"
	paramSecColumns    = "securities.columns"
	paramMarketColumns = "marketdata.columns"

	paramMetaValue = "off"
	paramOnlyValue = "securities,marketdata"
	// ACCRUEDINT	НКД	НКД на дату расчетов, в валюте расчетов
	// SHORTNAME	Кратк. наим.	Краткое наименование ценной бумаги
	// FACEUNIT	Валюта номинала	Валюта номинала
	// LOTVALUE	Номинал лота	Номинальная стоимость лота, в валюте номинала
	paramSecColumnsValue = "SECID,SHORTNAME,ACCRUEDINT,LOTVALUE"
	// LCURRENTPRICE	Текущая цена	Текущая цена
	paramMarketColumnsValue = "SECID,LCURRENTPRICE"

	fieldCouponDate  = "coupondate"
	fieldSecID       = "secid"
	fieldCouponValue = "value_rub"

	couponDateFormat = "2006-01-02"
)

type MoexData struct {
	Columns []string       `json:"columns"`
	Data    [][]any        `json:"data"`
	idx     map[string]int `json:"-"`
}

func (md *MoexData) BuildIndex() {
	secIDIdx := -1
	for i, columnName := range md.Columns {
		if strings.EqualFold(columnName, fieldSecID) {
			secIDIdx = i

			break
		}
	}

	if secIDIdx < 0 {
		return
	}

	mapIdx := map[string]int{}
	for i, data := range md.Data {
		if keyStr, ok := data[0].(string); ok {
			mapIdx[keyStr] = i
		}
	}

	md.idx = mapIdx
}

func (md *MoexData) Get(secID string) []any {
	if idx, ok := md.idx[secID]; ok {
		return md.Data[idx]
	}

	return nil
}

type MoexBondData struct {
	Securities MoexData `json:"securities"`
	MarketData MoexData `json:"marketdata"`
}

func (mbd *MoexBondData) BuildIndexes() {
	mbd.Securities.BuildIndex()
	mbd.MarketData.BuildIndex()
}

type MoexAPI struct {
	infoData *MoexBondData
	log      *slog.Logger
}

func NewMoexAPI(log *slog.Logger) *MoexAPI {
	return &MoexAPI{
		log: log.With(slog.String("struct", "MoexAPI")),
	}
}

/*
 * Info https://iss.moex.com/iss/engines/stock/markets/bonds/
 * https://iss.moex.com/iss/engines/stock/markets/bonds/boards/TQOB/securities.json?iss.meta=off&iss.only=securities,marketdata&securities.columns=SECID,SECNAME,PREVLEGALCLOSEPRICE&marketdata.columns=SECID,YIELD,DURATION"
 */

func (m *MoexAPI) loadInfo() error {
	reqURL, err := url.Parse(securitiesBasePath)
	if err != nil {
		return fmt.Errorf("cannot parse baseURL: %w", err)
	}

	params := url.Values{}
	params.Add(paramMeta, paramMetaValue)
	params.Add(paramOnly, paramOnlyValue)
	params.Add(paramSecColumns, paramSecColumnsValue)
	params.Add(paramMarketColumns, paramMarketColumnsValue)

	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot execure request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("not 200 request status: %d (%s)", resp.StatusCode, resp.Status)
	}

	data := MoexBondData{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("cannot decone json: %w", err)
	}

	data.BuildIndexes()
	m.infoData = &data

	return nil
}

func (m *MoexAPI) GetMarketInfo(secID string) (*domain.MarketInfo, error) {
	log := m.log.With(slog.String("op", "GetMarketInfo"))

	if m.infoData == nil {
		log.Info("Try load data")
		if err := m.loadInfo(); err != nil {
			log.Error("Cannot load moex data", slog.Any("error", err))

			return nil, err
		}
	}

	mi := domain.MarketInfo{}

	sdata := m.infoData.Securities.Get(secID)
	if sdata != nil {
		for i, column := range m.infoData.Securities.Columns {
			switch column {
			case "SECID":
				if strData, ok := sdata[i].(string); ok {
					mi.SECID = strData
				}
			case "SHORTNAME":
				if strData, ok := sdata[i].(string); ok {
					mi.Name = strData
				}
			case "ACCRUEDINT":
				if floatData, ok := sdata[i].(float64); ok {
					mi.CurrentNKD = floatData
				}
			case "LOTVALUE":
				if floatData, ok := sdata[i].(float64); ok {
					mi.LotValue = floatData
				}
			}
		}
	}

	mdata := m.infoData.MarketData.Get(secID)
	if mdata != nil {
		for i, column := range m.infoData.MarketData.Columns {
			switch column {
			case "LCURRENTPRICE":
				if floatData, ok := mdata[i].(float64); ok {
					mi.CurrentPrice = floatData
				}
			}
		}
	}

	return &mi, nil
}

type MoexCouponsData struct {
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
}

func (d *MoexCouponsData) GetColumnIndex(name string) int {
	for i, columnName := range d.Columns {
		if strings.EqualFold(name, columnName) {
			return i
		}
	}

	return -1
}

type MoexCoupons struct {
	Coupons MoexCouponsData `json:"coupons"`
}

// https://iss.moex.com/iss/statistics/engines/stock/markets/bonds/bondization/SU26247RMFS5.json?iss.meta=off
func (m *MoexAPI) GetCouponSchedule(secID string) ([]domain.CouponPayment, error) {
	log := m.log.With(slog.String("op", "GetMarketInfo"))

	reqURL, err := url.Parse(fmt.Sprintf(couponsURLtemplate, secID))
	if err != nil {
		return nil, fmt.Errorf("cannot parse baseURL: %w", err)
	}

	params := url.Values{}
	params.Add(paramMeta, paramMetaValue)

	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot execure request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("not 200 request status: %d (%s)", resp.StatusCode, resp.Status)
	}

	data := MoexCoupons{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		// if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("cannot decone json: %w", err)
	}

	coupons := make([]domain.CouponPayment, 0, len(data.Coupons.Data))
	idxSecID := data.Coupons.GetColumnIndex(fieldSecID)
	idxCouponDate := data.Coupons.GetColumnIndex(fieldCouponDate)
	idxCouponValue := data.Coupons.GetColumnIndex(fieldCouponValue)

	for _, couponData := range data.Coupons.Data {
		if !checkBounds(len(couponData), idxSecID, idxCouponDate, idxCouponValue) {
			return nil, fmt.Errorf("column index out of range")
		}

		rowSecID, ok := couponData[idxSecID].(string)
		if !ok {
			log.Error("cannot cast rowSecID to string", slog.String("secID", secID))

			return nil, fmt.Errorf("cannot cast rowSecID to string")
		}
		if !strings.EqualFold(secID, rowSecID) {
			log.Error("Coupon data secID is not equal", slog.String("secID", secID), slog.String("rowSecID", rowSecID))

			return nil, fmt.Errorf("coupon data secID is not equal")
		}

		rowCouponDate, ok := couponData[idxCouponDate].(string)
		if !ok {
			log.Error("cannot cast rowCouponDate to string", slog.String("secID", secID))

			return nil, fmt.Errorf("cannot cast rowCouponDate to string")
		}

		var coupon domain.CouponPayment
		coupon.Date, err = time.Parse(couponDateFormat, rowCouponDate)
		if err != nil {
			log.Error("Cannot convert rowCouponDate to time", slog.String("secID", secID), slog.Any("error", err))

			return nil, fmt.Errorf("cannot convert rowCouponDate to time: %w", err)
		}

		couponVal, ok := couponData[idxCouponValue].(float64)
		if !ok {
			log.Error("cannot convert coupon value to float32", slog.String("secID", secID))

			return nil, fmt.Errorf("cannot convert coupon value to float32")
		}

		coupon.Value = couponVal

		coupons = append(coupons, coupon)
	}

	return coupons, nil
}

func checkBounds(arrayLength int, indexes ...int) bool {
	for _, idx := range indexes {
		if idx < 0 || idx >= arrayLength {
			return false
		}
	}

	return true
}
