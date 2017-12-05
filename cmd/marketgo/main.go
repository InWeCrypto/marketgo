package main

import (
	"flag"

	"github.com/dynamicgo/aliyunlog"
	"github.com/dynamicgo/config"
	"github.com/dynamicgo/slf4go"
	"github.com/inwecrypto/marketgo"
	_ "github.com/lib/pq"
)

var logger = slf4go.Get("marketgo")
var configpath = flag.String("conf", "./marketgo.json", "market go  config file path")

func main() {

	flag.Parse()

	conf, err := config.NewFromFile(*configpath)

	if err != nil {
		logger.ErrorF("load neo config err , %s", err)
		return
	}

	factory, err := aliyunlog.NewAliyunBackend(conf)

	if err != nil {
		logger.ErrorF("create aliyun log backend err , %s", err)
		return
	}

	slf4go.Backend(factory)

	server := marketgo.NewMarketGo(conf)

	server.Run()
}
