package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	gecko "github.com/xiewei/gogo/v3"
	geckoTypes "github.com/xiewei/gogo/v3/types"
)

var (
	headers = []string{
		"id",
		"symbol",
		"name",
		"genesis_date",
		"market_cap",
		"market_rank",
		"change_one_year",
		"ath",
		"ath_date",
		"atl",
		"atl_date",
		"change_price",
		"atl_market_value",
		"atl_market_value_date",
		"ath_market_value",
		"ath_market_value_date",
		"change_market_value",
		"total_supply",
		"current_price",
	}
)

var (
	startTime, _ = time.Parse("2006-01-02 15:04:05", "2019-01-01 00:00:00")
	startUnix    = strconv.FormatInt(startTime.Unix(),10)
	endTime, _   = time.Parse("2006-01-02 15:04:05", "2021-12-31 23:59:59")
	endUnix      = strconv.FormatInt(endTime.Unix(),10)
	year = time.Date(2019,1,1,0,0,0,0,time.Local)
)

type Cryptocurrency struct {
	Id            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	MarketCap     float64 `json:"market_cap"`
	MarketRank    int16   `json:"market_rank"`
	ChangeOneYear float64 `json:"change_one_year"`
	ATH           float64 `json:"ath"`
	ATHDate       string  `json:"ath_date"`
	TotalSupply   float64 `json:"total_supply"`
	CurrentPrice  float64 `json:"current_price"`

	ATL     float64 `json:"atl"`
	ATLDate string  `json:"atl_date"`

	ChangePrice float64 `json:"change_price"` // (ath - atl) / atl * 100

	ATLMarketValue float32 `json:"atl_market_value"`
	ATLMarketDate string `json:"atl_market_date"`
	ATHMarketValue float32 `json:"ath_market_value"`
	ATHMarketDate string `json:"ath_market_date"`
	ChangeMarketCap float32 `json:"change_market_cap"` // (ath_market_value - atl_market_value)/atl

	GenesisDate string `json:"genesis_date"`
}

func (c Cryptocurrency) ToStrings() []string {
	return []string{
		c.Id,
		c.Symbol,
		c.Name,
		c.GenesisDate,
		fmt.Sprintf("%.0f", c.MarketCap),
		fmt.Sprintf("%.0d", c.MarketRank),
		fmt.Sprintf("%.2f", c.ChangeOneYear),
		fmt.Sprintf("%.12f", c.ATH),
		c.ATHDate,

		fmt.Sprintf("%.12f", c.ATL),
		c.ATLDate,

		fmt.Sprintf("%.2f", c.ChangePrice),

		fmt.Sprintf("%.2f", c.ATHMarketValue),
		c.ATHMarketDate,
		fmt.Sprintf("%.2f", c.ATLMarketValue),
		c.ATLMarketDate,
		fmt.Sprintf("%.2f", c.ChangeMarketCap),

		fmt.Sprintf("%.0f", c.TotalSupply),
		fmt.Sprintf("%f", c.CurrentPrice),
	}
}

type Compare struct {
	List []geckoTypes.ChartItem
}

type HL struct {
	Min float32
	Max float32
	MinDate string
	MaxDate string
}

func (c Compare) MinMax() HL{
	var list = make([]geckoTypes.ChartItem, 0)
	for _, i := range c.List {
		if i[1] !=0 {
			list = append(list, i)
		}
	}
	var (
		min float32 = 0
		max float32 = 0
		minDate string
		maxDate string
	)
	min, max = list[0][1], list[0][1]
	minDate = time.Unix(int64(list[0][0]/1e3),0).Format("2006-01-02")
	maxDate = minDate
	for _, i := range list {
		if i[1] < min {
			min = i[1]
			minDate = time.Unix(int64(i[0]/1e3),0).Format("2006-01-02")
		}
		if i[1] > max {
			max = i[1]
			maxDate = time.Unix(int64(i[0]/1e3),0).Format("2006-01-02")
		}
	}
	return HL{
		Min:     min,
		Max:     max,
		MinDate: minDate,
		MaxDate: maxDate,
	}
}
func main() {
	cg := gecko.NewClient(nil)
	vsCurrency := "usd"
	perPage := 100
	page := 1
	sparkline := false
	pcp := geckoTypes.PriceChangePercentageObject
	priceChangePercentage := []string{pcp.PCP1y}
	order := geckoTypes.OrderTypeObject.MarketCapDesc
	market, err := cg.CoinsMarket(vsCurrency, nil, order, perPage, page, sparkline, priceChangePercentage)
	if err != nil {
		log.Fatal(err)
	}
	var results = make([]Cryptocurrency, 0)
	for _, i := range *market {
		var one float64
		if i.PriceChangePercentage1yInCurrency != nil {
			one = *i.PriceChangePercentage1yInCurrency
		}
		athDate, _ := time.Parse(time.RFC3339, i.ATHDate)
		atlDate, _ := time.Parse(time.RFC3339, i.ATLDate)
		crypto := Cryptocurrency{
			Id:            i.ID,
			Symbol:        i.Symbol,
			Name:          i.Name,
			MarketCap:     i.MarketCap,
			MarketRank:    i.MarketCapRank,
			ChangeOneYear: one,
			ATH:           i.ATH,
			ATHDate:       i.ATHDate,
			ATL:           i.ATL,
			ATLDate:       i.ATLDate,
			TotalSupply:   i.TotalSupply,
			CurrentPrice:  i.CurrentPrice,
		}
		if athDate.After(year) && atlDate.After(year) {
			crypto.ChangePrice = ((i.ATH - i.ATL) / i.ATL) * 100
		}

		marketRange, _ := cg.CoinsIDMarketChartRage(i.ID, vsCurrency, startUnix, endUnix)
		if marketRange != nil {
			marketValue := *marketRange.MarketCaps
			hl := Compare{List: marketValue}.MinMax()
			crypto.ATLMarketValue = hl.Min
			crypto.ATHMarketValue = hl.Max
			crypto.ATLMarketDate = hl.MinDate
			crypto.ATHMarketDate = hl.MaxDate
			crypto.ChangeMarketCap = (crypto.ATHMarketValue - crypto.ATLMarketValue)/crypto.ATLMarketValue * 100
		}

		coin, _ := cg.CoinsID(i.ID, false, false, true, false, false, true)
		if coin != nil && coin.MarketData != nil {
			marketData := coin.MarketData
			if crypto.TotalSupply == 0 {
				if marketData.TotalSupply != nil {
					crypto.TotalSupply = *marketData.TotalSupply
				}
			}
			if crypto.ChangeOneYear == 0 {
				crypto.ChangeOneYear = marketData.PriceChangePercentage1y
				if crypto.ChangeOneYear == 0 {
					crypto.ChangeOneYear = marketData.PriceChangePercentage200d
				}
			}
			crypto.GenesisDate = coin.GenesisDate
		}
		results = append(results, crypto)
	}
	f, err := os.Create("crypto5.csv")
	if os.IsExist(err) {
		f, _ = os.Open("crypto5.csv")
	}
	w := csv.NewWriter(f)
	w.Write(headers)
	for _, i := range results {
		record := i.ToStrings()
		fmt.Println(record)
		err = w.Write(record)
		if err != nil {
			fmt.Println(err)
		}
	}
	w.Flush()
}
