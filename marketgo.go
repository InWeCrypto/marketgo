package marketgo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dynamicgo/config"
	"github.com/dynamicgo/slf4go"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

// Provider market data provider
type Provider interface {
	Open(symbol string, currency string, interval string, o chan []*Kline) bool
}

// Kline .
type Kline struct {
	StartTime  uint          `json:"time"`
	EndTime    uint          `json:"end_time"`
	MinPrice   string        `json:"min_price"`
	MaxPrice   string        `json:"max_price"`
	OpenPrice  string        `json:"opened_price"`
	ClosePrice string        `json:"closed_price"`
	Volume     string        `json:"volume"`
	Source     string        `json:"-"`
	Key        string        `json:"-"`
	Duration   time.Duration `json:"-"`
}

// MarketGo market service
type MarketGo struct {
	slf4go.Logger
	engine    *gin.Engine
	conf      *config.Config
	providers map[string]Provider
	redc      *redis.Client
	klines    chan []*Kline
}

// NewMarketGo create new marketgo service
func NewMarketGo(conf *config.Config) *MarketGo {

	cachedsize := conf.GetInt64("marketgo.cachedsize", 100)

	if !config.GetBool("marketgo.debug", false) {
		gin.SetMode(gin.ReleaseMode)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     conf.GetString("marketgo.redis.address", "localhost:6379"),
		Password: conf.GetString("marketgo.redis.password", "xxxxxx"), // no password set
		DB:       int(conf.GetInt64("marketgo.redis.db", 4)),          // use default DB
	})

	engine := gin.New()
	engine.Use(gin.Recovery())

	marketGo := &MarketGo{
		Logger:    slf4go.Get("neo-order-service"),
		engine:    engine,
		redc:      client,
		providers: make(map[string]Provider),
		klines:    make(chan []*Kline, cachedsize),
	}

	marketGo.makeProviders(conf)
	marketGo.makeRouters()

	go marketGo.flushRedis()

	return marketGo
}

func (marketGo *MarketGo) flushRedis() {
	for klines := range marketGo.klines {
		if len(klines) == 0 {
			continue
		}

		marketGo.DebugF("flush klines %s ", klines[0].Key)

		data, err := json.Marshal(klines)

		if err != nil {
			marketGo.ErrorF("flush klines %s err,%s", klines[0].Key, err)
			continue
		}

		_, err = marketGo.redc.Set(klines[0].Key, data, klines[0].Duration).Result()

		if err != nil {
			marketGo.ErrorF("flush klines %s err,%s", klines[0].Key, err)
			continue
		}
	}
}

func (marketGo *MarketGo) makeProviders(conf *config.Config) {
	marketGo.providers["binance"] = NewBinanceMarket(conf)
}

func (marketGo *MarketGo) makeRouters() {
	marketGo.engine.GET("/kline", func(ctx *gin.Context) {
		symbol := ctx.Query("symbol")
		interval := ctx.Query("interval")
		currency := ctx.Query("currency")
		klines, err := marketGo.kline(symbol, currency, interval)

		marketGo.DebugF("kline get %s %s %s", symbol, currency, interval)

		if symbol == "" {
			marketGo.ErrorF("invalid symbol for kline request")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unknown symbol"})
			return
		}

		if currency == "" {
			marketGo.ErrorF("invalid currency for kline request")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unknown currency"})
			return
		}

		if interval == "" {
			marketGo.ErrorF("invalid interval for kline request")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "unknown interval"})
			return
		}

		if err != nil {
			marketGo.ErrorF("get klines %s %s", symbol, interval)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, klines)
	})
}

func (marketGo *MarketGo) kline(symbol string, currency string, interval string) ([]*Kline, error) {
	key := fmt.Sprintf("%s_%s_%s", strings.ToUpper(symbol), strings.ToUpper(currency), interval)
	jsondata, err := marketGo.redc.Get(key).Result()

	marketGo.DebugF("%s query cached data", key)

	if err != nil {
		if err == redis.Nil {
			marketGo.DebugF("%s cached miss", key)
			for _, provider := range marketGo.providers {
				if provider.Open(symbol, currency, interval, marketGo.klines) {
					break
				}
			}
			return make([]*Kline, 0), nil
		}

		return nil, err
	}

	var klines []*Kline

	err = json.Unmarshal([]byte(jsondata), &klines)

	return klines, err
}

// Run .
func (marketGo *MarketGo) Run() error {
	return marketGo.engine.Run(config.GetString("marketgo.laddr", ":5000"))
}
