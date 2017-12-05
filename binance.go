package marketgo

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/sling"
	"github.com/dynamicgo/config"
	"github.com/dynamicgo/slf4go"
)

// BinanceMarket .
type BinanceMarket struct {
	slf4go.Logger
	klineapi string
	symbols  sync.Map
}

// BinanceKlineParam .
type BinanceKlineParam struct {
	Symbol   string `url:"symbol"`
	Interval string `url:"interval"`
}

// BinanceError .
type BinanceError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// NewBinanceMarket create new binance market cache client
func NewBinanceMarket(conf *config.Config) *BinanceMarket {

	market := &BinanceMarket{
		Logger:   slf4go.Get("binance"),
		klineapi: conf.GetString("marketgo.binance.klines", "https://www.binance.com/api/v1/klines"),
	}

	return market
}

// Open .
func (market *BinanceMarket) Open(symbol string, currency string, interval string, o chan []*Kline) bool {

	key := fmt.Sprintf("%s_%s_%s", symbol, currency, interval)

	if _, ok := market.symbols.LoadOrStore(key, key); ok {
		return true
	}

	go market.loopFetch(symbol, currency, interval, o)

	return true
}

func (market *BinanceMarket) loopFetch(symbol string, currency string, interval string, o chan []*Kline) {

	duration, ok := durations[interval]

	if !ok {
		market.WarnF("unknown interval %s", duration)
		return
	}

	ticker := time.NewTicker(duration)

	param := &BinanceKlineParam{
		Symbol:   fmt.Sprintf("%s%s", strings.ToUpper(symbol), strings.ToUpper(currency)),
		Interval: interval,
	}

	key := fmt.Sprintf("%s_%s_%s", strings.ToUpper(symbol), strings.ToUpper(currency), interval)

	market.doFetch(key, param, duration, o)

	for _ = range ticker.C {
		market.doFetch(key, param, duration, o)
	}
}

func (market *BinanceMarket) doFetch(key string, param *BinanceKlineParam, duration time.Duration, o chan []*Kline) {
	market.DebugF("start sync klines %s", key)

	client := sling.New()
	request, err := client.Get(market.klineapi).QueryStruct(param).Request()

	if err != nil {
		market.ErrorF("create kline http request err %s", err)
		return
	}

	var klines [][]interface{}
	var binanceError BinanceError

	market.DebugF("binance kline request %s", request.URL)

	response, err := client.Do(request, &klines, &binanceError)

	if err != nil {
		market.ErrorF("get kline err %s", err)
		return
	}

	if response.StatusCode != http.StatusOK {
		market.ErrorF("get kline err %d %s", binanceError.Code, binanceError.Message)
		return
	}

	var result []*Kline

	for _, kline := range klines {
		result = append(result, &Kline{
			StartTime:  uint(kline[0].(float64)),
			EndTime:    uint(kline[6].(float64)),
			MinPrice:   kline[3].(string),
			MaxPrice:   kline[2].(string),
			OpenPrice:  kline[1].(string),
			ClosePrice: kline[4].(string),
			Volume:     kline[5].(string),
			Key:        key,
			Duration:   duration,
		})
	}

	o <- result
}
