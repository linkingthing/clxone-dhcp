package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	restserver "github.com/linkingthing/clxone-dhcp/server"
)

var (
	configFile string
	ip         string
	port       int
)

func main() {
	flag.StringVar(&configFile, "c", "controller.conf", "configure file path")
	flag.StringVar(&ip, "ip", "127.0.0.1", "server port")
	flag.IntVar(&port, "p", 58221, "server port")
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
	conf.Server.IP = ip
	conf.Server.Port = port

	db.RegisterResources(dhcp.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		log.Fatalf("init db failed: %s", err.Error())
	}

	conn, err := grpc.Dial(conf.DHCPAgent.GrpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dail grpc failed: %s", err.Error())
	}
	defer conn.Close()
	grpcclient.NewDhcpClient(conn)

	server, err := restserver.NewServer()
	if err != nil {
		log.Fatalf("new server failed: %s", err.Error())
	}

	server.RegisterHandler(restserver.HandlerRegister(dhcp.RegisterHandler))

	if err := server.Run(conf); err != nil {
		log.Fatalf("server run failed: %s", err.Error())
	}
}
