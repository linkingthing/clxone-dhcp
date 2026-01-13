package main

import (
	"flag"
	"fmt"

	"github.com/linkingthing/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/alarm"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp"
	"github.com/linkingthing/clxone-dhcp/pkg/metric"
	"github.com/linkingthing/clxone-dhcp/pkg/transport"
	restserver "github.com/linkingthing/clxone-dhcp/server"
)

var (
	configFile                 string
	logLevel                   string
	version, commit, buildTime string
)

func main() {
	flag.StringVar(&configFile, "c", "clxone-dhcp.conf", "configure file path")
	flag.StringVar(&logLevel, "l", "info", "log level")
	flag.Parse()

	fmt.Printf("build version:%s commit:%s time:%s\n", version, commit, buildTime)
	log.InitLogger(log.LogLevel(logLevel))

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s", err.Error())
	}

	if err := initServer(conf); err != nil {
		log.Fatalf("init server failed: %s", err.Error())
	}
}

func initServer(conf *config.DHCPConfig) error {
	db.RegisterResources(dhcp.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		return fmt.Errorf("init db failed: %s", err.Error())
	}

	alarm.Init(conf)
	server, err := restserver.NewServer()
	if err != nil {
		return fmt.Errorf("new server failed: %s", err.Error())
	}

	if err := server.RegisterHandler(restserver.HandlerRegister(dhcp.RegisterApi)); err != nil {
		return fmt.Errorf("register dhcp handler failed: %s", err.Error())
	}
	if err := transport.RegisterTransport(conf); err != nil {
		return fmt.Errorf("register transport failed: %s", err.Error())
	}
	if err := server.RegisterHandler(restserver.HandlerRegister(metric.RegisterApi)); err != nil {
		return fmt.Errorf("register metric handler failed: %s", err.Error())
	}
	return server.Run(conf)
}
