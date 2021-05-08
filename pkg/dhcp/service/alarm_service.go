package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcpclient"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/alarm"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	ThresholdTopic      = "threshold"
	ThresholdDhcpTopic  = "threshold_dhcp"
	RegisterThreshold   = "register_threshold"
	UpdateThreshold     = "update_threshold"
	DeRegisterThreshold = "de_register_threshold"
	IllegalDhcpAlarm    = "illegal_dhcp_alarm"
)

var globalAlarmService *AlarmService
var once sync.Once

type AlarmService struct {
	DhcpThreshold *alarm.RegisterThreshold
}

func NewAlarmService() *AlarmService {
	once.Do(func() {
		globalAlarmService = &AlarmService{
			DhcpThreshold: &alarm.RegisterThreshold{
				BaseThreshold: &alarm.BaseThreshold{
					Name:  alarm.ThresholdName_illegalDhcp,
					Level: alarm.ThresholdLevel_critical,
					Type:  alarm.ThresholdType_trigger,
				},
				Value:    0,
				SendMail: false,
			},
		}
	})
	return globalAlarmService
}

func (a *AlarmService) RegisterThresholdToKafka(key string, threshold *alarm.RegisterThreshold) error {
	data, err := proto.Marshal(threshold)
	if err != nil {
		return fmt.Errorf("register threshold mashal failed: %s ", err.Error())
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:   config.GetConfig().Kafka.Addr,
		Topic:     ThresholdTopic,
		BatchSize: 1,
		Dialer: &kafka.Dialer{
			Timeout:   time.Second * 10,
			DualStack: true,
			KeepAlive: time.Second * 5},
	})

	defer w.Close()

	err = w.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: data,
		},
	)
	if err != nil {
		logrus.Error(err)
	}
	return err
}

func (a *AlarmService) SendEventWithIllegalDHCP(dhcpServers []*dhcpclient.DHCPServer,
	illegalAlarm alarm.IllegalDhcpAlarm) (err error) {
	if len(dhcpServers) == 0 {
		return
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:   config.GetConfig().Kafka.Addr,
		Topic:     ThresholdDhcpTopic,
		BatchSize: 1,
		Dialer: &kafka.Dialer{
			Timeout:   time.Second * 10,
			DualStack: true,
			KeepAlive: time.Second * 5},
	})

	defer w.Close()

	for _, dhcpServer := range dhcpServers {
		ip := dhcpServer.IPv4
		if ip == "" {
			ip = dhcpServer.IPv6
		}

		illegalAlarm.IllegalDhcpIp = ip
		illegalAlarm.IllegalDhcpMac = dhcpServer.Mac

		data, err := proto.Marshal(&illegalAlarm)
		if err != nil {
			return fmt.Errorf("register threshold mashal failed: %s ", err.Error())
		}
		err = w.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(IllegalDhcpAlarm),
				Value: data,
			},
		)
		if err != nil {
			logrus.Error(err)
		}
	}
	return nil
}
