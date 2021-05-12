package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp"
	restserver "github.com/linkingthing/clxone-dhcp/server"
)

var (
	configFile string
	ip         string
	port       int
)

func main() {
	flag.StringVar(&configFile, "c", "controller.conf", "configure file path")
	flag.Parse()

	log.InitLogger(log.Debug)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s", err.Error())
	}

	db.RegisterResources(dhcp.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		log.Fatalf("init db failed: %s", err.Error())
	}

	// conn := grpcclient.NewDhcpAgentClient()
	// defer conn.Close()

	server, err := restserver.NewServer()
	if err != nil {
		log.Fatalf("new server failed: %s", err.Error())
	}

	server.RegisterHandler(restserver.HandlerRegister(dhcp.RegisterHandler))

	if err := server.Run(conf); err != nil {
		log.Fatalf("server run failed: %s", err.Error())
	}
}
