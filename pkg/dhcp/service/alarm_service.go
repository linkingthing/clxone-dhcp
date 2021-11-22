package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/linkingthing/cement/log"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"github.com/linkingthing/clxone-dhcp/config"
	pbalarm "github.com/linkingthing/clxone-dhcp/pkg/proto/alarm"
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
	DhcpThreshold *pbalarm.RegisterThreshold
	LpsThreshold  *pbalarm.RegisterThreshold
	kafkaWriter   *kafka.Writer
}

func NewAlarmService() *AlarmService {
	onceAlarmService.Do(func() {
		globalAlarmService = &AlarmService{
			DhcpThreshold: &pbalarm.RegisterThreshold{
				BaseThreshold: &pbalarm.BaseThreshold{
					Name:  pbalarm.ThresholdName_illegalDhcp,
					Level: pbalarm.ThresholdLevel_major,
					Type:  pbalarm.ThresholdType_trigger,
				},
				Value:    0,
				SendMail: false,
				Enabled:  true,
			},
			LpsThreshold: &pbalarm.RegisterThreshold{
				BaseThreshold: &pbalarm.BaseThreshold{
					Name:  pbalarm.ThresholdName_lps,
					Level: pbalarm.ThresholdLevel_critical,
					Type:  pbalarm.ThresholdType_values,
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

func (a *AlarmService) RegisterThresholdToKafka(key string, threshold *pbalarm.RegisterThreshold) error {
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

func (a *AlarmService) HandleUpdateThresholdEvent(topic string, updateFunc func(*pbalarm.UpdateThreshold)) {
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
			var req pbalarm.UpdateThreshold
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("handle update threshold when unmarshal alarm %s threahold failed: %s",
					message.Key, err.Error())
			} else {
				updateFunc(&req)
			}
		}
	}
}

func (a *AlarmService) UpdateDhcpThresHold(update *pbalarm.UpdateThreshold) {
	if update.Name == pbalarm.ThresholdName_illegalDhcp {
		a.DhcpThreshold = &pbalarm.RegisterThreshold{
			Value:    update.Value,
			SendMail: update.SendMail,
			Enabled:  update.Enabled,
		}
	}
}

func (a *AlarmService) UpdateLpsThresHold(update *pbalarm.UpdateThreshold) {
	if update.Name != pbalarm.ThresholdName_lps {
		a.LpsThreshold = &pbalarm.RegisterThreshold{
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
