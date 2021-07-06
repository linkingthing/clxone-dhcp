package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/protobuf/proto"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/alarm"
)

const (
	ThresholdTopic     = "threshold"
	ThresholdDhcpTopic = "threshold_dhcp"

	RegisterThreshold   = "register_threshold"
	UpdateThreshold     = "update_threshold"
	DeRegisterThreshold = "de_register_threshold"
)

const (
	AlarmTopic = "alarm"

	AlarmKeyIllegalDhcp = "illegal_dhcp_alarm"
	AlarmKeyLps         = "lps_alarm"
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
		globalAlarmService = &AlarmService{
			DhcpThreshold: &alarm.RegisterThreshold{
				BaseThreshold: &alarm.BaseThreshold{
					Name:  alarm.ThresholdName_illegalDhcp,
					Level: alarm.ThresholdLevel_major,
					Type:  alarm.ThresholdType_trigger,
				},
				Value:    0,
				SendMail: false,
				Enabled:  true,
			},
			LpsThreshold: &alarm.RegisterThreshold{
				BaseThreshold: &alarm.BaseThreshold{
					Name:  alarm.ThresholdName_lps,
					Level: alarm.ThresholdLevel_critical,
					Type:  alarm.ThresholdType_values,
				},
				Value:    3000,
				SendMail: false,
				Enabled:  true,
			},
			kafkaWriter: kafka.NewWriter(kafka.WriterConfig{
				Brokers:   config.GetConfig().Kafka.Addrs,
				BatchSize: 1,
				Dialer: &kafka.Dialer{
					Timeout:   time.Second * 10,
					DualStack: true,
					KeepAlive: time.Second * 5},
			}),
		}
	})
	return globalAlarmService
}

func (a *AlarmService) RegisterThresholdToKafka(key string, threshold *alarm.RegisterThreshold) error {
	data, err := proto.Marshal(threshold)
	if err != nil {
		return fmt.Errorf("register threshold mashal failed: %s ", err.Error())
	}

	return a.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: data,
			Topic: ThresholdTopic,
		},
	)
}

func (a *AlarmService) HandleUpdateThresholdEvent(topic string, updateFunc func(*alarm.UpdateThreshold)) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.GetConfig().Kafka.Addrs,
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
			log.Warnf("read update threshold message from kafka failed: %s", err.Error())
			continue
		}

		switch string(message.Key) {
		case UpdateThreshold:
			var req alarm.UpdateThreshold
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("handle update threshold when unmarshal alarm %s threahold failed: %s",
					message.Key, err.Error())
			} else {
				updateFunc(&req)
			}
		}
	}
}

func (a *AlarmService) UpdateDhcpThresHold(update *alarm.UpdateThreshold) {
	if update.Name == alarm.ThresholdName_illegalDhcp {
		a.DhcpThreshold = &alarm.RegisterThreshold{
			Value:    update.Value,
			SendMail: update.SendMail,
			Enabled:  update.Enabled,
		}
	}
}

func (a *AlarmService) UpdateLpsThresHold(update *alarm.UpdateThreshold) {
	if update.Name != alarm.ThresholdName_lps {
		a.LpsThreshold = &alarm.RegisterThreshold{
			Value:    update.Value,
			SendMail: update.SendMail,
			Enabled:  update.Enabled,
		}
	}
}

func (a *AlarmService) SendEventWithValues(key string, param proto.Message) {
	data, err := proto.Marshal(param)
	if err != nil {
		log.Errorf("send alarm %s when marshal param %s failed: %s", key, param, err.Error())
		return
	}

	if err := a.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Topic: AlarmTopic,
			Key:   []byte(key),
			Value: data,
		},
	); err != nil {
		log.Errorf("send alarm %s to kafka failed: %s", key, err.Error())
	}
}
