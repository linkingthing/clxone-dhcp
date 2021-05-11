package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/alarm"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	ThresholdTopic        = "threshold"
	ThresholdIpamTopic    = "threshold_ipam"
	ThresholdDnsTopic     = "threshold_dns"
	ThresholdDhcpTopic    = "threshold_dhcp"
	ThresholdLpsTopic     = "threshold_lps"
	ThresholdMonitorTopic = "threshold_monitor"

	RegisterThreshold   = "register_threshold"
	UpdateThreshold     = "update_threshold"
	DeRegisterThreshold = "de_register_threshold"
)

const (
	AlarmTopic = "alarm"

	NodeAlarm           = "node_alarm"
	NodeHaAlarm         = "node_ha_alarm"
	MetricValueAlarm    = "metric_value_alarm"
	SubnetRadioAlarm    = "subnet_radio_alarm"
	ConflictIpAlarm     = "conflict_ip_alarm"
	ConflictSubnetAlarm = "conflict_subnet_alarm"
	IllegalDhcpAlarm    = "illegal_dhcp_alarm"
	IpBaseLineAlarm     = "ip_base_line_alarm"
	AlarmResponse       = "alarm_response"
)

var globalAlarmService *AlarmService
var onceAlarmService sync.Once

type AlarmService struct {
	DhcpThreshold *alarm.RegisterThreshold
	LpsThreshold  *alarm.RegisterThreshold
}

func NewAlarmService() *AlarmService {
	onceAlarmService.Do(func() {
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
			LpsThreshold: &alarm.RegisterThreshold{
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

func (a *AlarmService) HandleUpdateThresholdEvent(topic string, updateFunc func(*alarm.UpdateThreshold)) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.GetConfig().Kafka.Addr,
		Topic:          topic,
		MinBytes:       10,
		MaxBytes:       10e6,
		SessionTimeout: time.Second * 10,
		Dialer: &kafka.Dialer{
			Timeout:   time.Second * 10,
			DualStack: true,
			KeepAlive: time.Second * 5},
	})
	defer r.Close()

	for {
		ctx := context.Background()
		m, err := r.FetchMessage(ctx)
		if err != nil {
			break
		}

		switch string(m.Key) {
		case UpdateThreshold:
			var req alarm.UpdateThreshold
			if err := proto.Unmarshal(m.Value, &req); err != nil {
				logrus.Error(err)
			}
			updateFunc(&req)
		}

		if err := r.CommitMessages(ctx, m); err != nil {
			log.Fatal("failed to commit messages:", err)
		}
	}
}

func (a *AlarmService) UpdateDhcpThresHold(update *alarm.UpdateThreshold) {
	a.DhcpThreshold = &alarm.RegisterThreshold{
		Value:    update.Value,
		SendMail: update.SendMail,
	}
}

func (a *AlarmService) UpdateLpsThresHold(update *alarm.UpdateThreshold) {
	a.LpsThreshold = &alarm.RegisterThreshold{
		Value:    update.Value,
		SendMail: update.SendMail,
	}
}

func (a *AlarmService) SendEventWithValues(param proto.Message) {
	data, err := proto.Marshal(param)
	if err != nil {
		logrus.Error(err)
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
