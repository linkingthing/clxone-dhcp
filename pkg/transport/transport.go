package transport

import (
	"log"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

func RegisterTransport(conf *config.DHCPConfig) error {
	if err := service.NewAlarmService(conf); err != nil {
		log.Fatalf("new alarm service failed: %s", err.Error())
	}
	return nil
}
