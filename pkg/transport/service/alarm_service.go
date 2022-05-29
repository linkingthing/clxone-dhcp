package service

import (
	"github.com/linkingthing/clxone-utils/alarm"
	alarmProto "github.com/linkingthing/clxone-utils/alarm/proto"

	"github.com/linkingthing/clxone-dhcp/config"
)

var baseThreshold = []*alarm.Threshold{
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_illegalDhcp,
			Level: alarmProto.ThresholdLevel_major,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_lps,
			Level: alarmProto.ThresholdLevel_critical,
			Type:  alarmProto.ThresholdType_values,
		},
		Value:    3000,
		SendMail: false,
		Enabled:  true,
	},
}

var globalAlarm *alarm.Alarm

func GetAlarmService() *alarm.Alarm {
	return globalAlarm
}

func NewAlarmService(conf *config.DHCPConfig) error {
	tmpAlarm, err := alarm.RegisterAlarm(alarm.KafkaConf{
		Addresses: conf.Kafka.Addrs,
		Username:  conf.Kafka.Username,
		Password:  conf.Kafka.Password,
		Topic:     alarm.ThresholdDhcpTopic,
		GroupId:   conf.Kafka.GroupUpdateThresholdEvent,
	}, baseThreshold...)
	if err != nil {
		return err
	}
	globalAlarm = tmpAlarm
	return nil
}
