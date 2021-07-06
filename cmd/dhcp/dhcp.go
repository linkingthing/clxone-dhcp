package main

import (
	"flag"

	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp"
	"github.com/linkingthing/clxone-dhcp/pkg/metric"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	restserver "github.com/linkingthing/clxone-dhcp/server"
)

var (
	configFile string
	ip         string
	port       int
)

func main() {
	flag.StringVar(&configFile, "c", "clxone-dhcp.conf", "configure file path")
	flag.Parse()

	log.InitLogger(log.Info)

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s", err.Error())
	}

	db.RegisterResources(dhcp.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		log.Fatalf("init db failed: %s", err.Error())
	}

	server, err := restserver.NewServer()
	if err != nil {
		log.Fatalf("new server failed: %s", err.Error())
	}

	if err := server.RegisterHandler(restserver.HandlerRegister(dhcp.RegisterHandler)); err != nil {
		log.Fatalf("register dhcp handler failed: %s", err.Error())
	}

	if err := server.RegisterHandler(restserver.HandlerRegister(metric.RegisterHandler)); err != nil {
		log.Fatalf("register metric handler failed: %s", err.Error())
	}

	defer pb.CloseConns()
	if err := server.Run(conf); err != nil {
		log.Fatalf("server run failed: %s", err.Error())
	}
}
