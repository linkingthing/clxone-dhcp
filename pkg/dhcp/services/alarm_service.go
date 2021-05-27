package services

import (
	"context"
	"fmt"
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
	kafkaWriter   *kafka.Writer
}

func NewAlarmService() *AlarmService {
	onceAlarmService.Do(func() {
		globalAlarmService = &AlarmService{}
		{
			dhcpThreshold := &alarm.RegisterThreshold{
				BaseThreshold: &alarm.BaseThreshold{
					Name:  alarm.ThresholdName_illegalDhcp,
					Level: alarm.ThresholdLevel_major,
					Type:  alarm.ThresholdType_trigger,
				},
				Value:    0,
				SendMail: false,
			}
			globalAlarmService.DhcpThreshold = dhcpThreshold
		}
		{
			lpsThreshold := &alarm.RegisterThreshold{
				BaseThreshold: &alarm.BaseThreshold{
					Name:  alarm.ThresholdName_lps,
					Level: alarm.ThresholdLevel_critical,
					Type:  alarm.ThresholdType_values,
				},
				Value:    3000,
				SendMail: false,
			}
			globalAlarmService.LpsThreshold = lpsThreshold
		}
		{
			w := kafka.NewWriter(kafka.WriterConfig{
				Brokers:   config.GetConfig().Kafka.Addr,
				BatchSize: 1,
				Dialer: &kafka.Dialer{
					Timeout:   time.Second * 10,
					DualStack: true,
					KeepAlive: time.Second * 5},
			})
			globalAlarmService.kafkaWriter = w
		}
	})
	return globalAlarmService
}

func (a *AlarmService) RegisterThresholdToKafka(key string, threshold *alarm.RegisterThreshold) error {
	data, err := proto.Marshal(threshold)
	if err != nil {
		return fmt.Errorf("register threshold mashal failed: %s ", err.Error())
	}

	err = a.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: data,
			Topic: ThresholdTopic,
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
		GroupID:        config.GetConfig().Kafka.GroupUpdateThresholdEvent,
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
		message, err := r.ReadMessage(ctx)
		if err != nil {
			break
		}

		switch string(message.Key) {
		case UpdateThreshold:
			var req alarm.UpdateThreshold
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				logrus.Error(err)
			}
			updateFunc(&req)
		}
	}
}

func (a *AlarmService) UpdateDhcpThresHold(update *alarm.UpdateThreshold) {
	if update.Name != alarm.ThresholdName_illegalDhcp {
		return
	}
	a.DhcpThreshold = &alarm.RegisterThreshold{
		Value:    update.Value,
		SendMail: update.SendMail,
	}
}

func (a *AlarmService) UpdateLpsThresHold(update *alarm.UpdateThreshold) {
	if update.Name != alarm.ThresholdName_lps {
		return
	}
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

	err = a.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Topic: AlarmTopic,
			Key:   []byte(IllegalDhcpAlarm),
			Value: data,
		},
	)
	if err != nil {
		logrus.Error(err)
	}
}
