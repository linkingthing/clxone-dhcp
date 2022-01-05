package api

import (
	alarmutil "github.com/linkingthing/clxone-utils/alarm"
	pbutil "github.com/linkingthing/clxone-utils/alarm/proto"

	"github.com/linkingthing/clxone-dhcp/config"
)

var globalAlarm *alarmutil.Alarm

func GetAlarmService() *alarmutil.Alarm {
	return globalAlarm
}

func NewAlarmService(conf *config.DHCPConfig) error {
	alarm, err := alarmutil.RegisterAlarm(alarmutil.KafkaConf{
		Addresses: conf.Kafka.Addrs,
		Topic:     alarmutil.ThresholdDhcpTopic,
		GroupId:   conf.Kafka.GroupUpdateThresholdEvent,
	}, []*alarmutil.Threshold{
		{
			BaseThreshold: &pbutil.BaseThreshold{
				Name:  pbutil.ThresholdName_illegalDhcp,
				Level: pbutil.ThresholdLevel_major,
				Type:  pbutil.ThresholdType_trigger,
			},
			Value:    0,
			SendMail: false,
			Enabled:  true,
		},
		{
			BaseThreshold: &pbutil.BaseThreshold{
				Name:  pbutil.ThresholdName_lps,
				Level: pbutil.ThresholdLevel_critical,
				Type:  pbutil.ThresholdType_values,
			},
			Value:    3000,
			SendMail: false,
			Enabled:  true,
		},
	}...)
	if err != nil {
		return err
	}
	globalAlarm = alarm
	return nil
}
