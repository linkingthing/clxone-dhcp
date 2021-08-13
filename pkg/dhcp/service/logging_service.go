package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/logging"
)

const (
	LoggingTopic   = "logging"
	LoggingRequest = "logging_request"
)

var globalLoggingService *LoggingService
var onceLoggingService sync.Once

type LoggingService struct {
	kafakWrite *kafka.Writer
}

func NewLoggingService() *LoggingService {
	onceLoggingService.Do(func() {
		globalLoggingService = &LoggingService{}
		w := kafka.NewWriter(kafka.WriterConfig{
			Brokers:   config.GetConfig().Kafka.Addrs,
			Topic:     LoggingTopic,
			BatchSize: 1,
			Dialer: &kafka.Dialer{
				Timeout:   time.Second * 10,
				DualStack: true,
				KeepAlive: time.Second * 5},
		})
		globalLoggingService.kafakWrite = w
	})
	return globalLoggingService
}

func (a *LoggingService) Log(req *logging.LoggingRequest) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("register threshold mashal failed: %s ", err.Error())
	}

	return a.kafakWrite.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(LoggingRequest),
			Value: data,
		},
	)
}