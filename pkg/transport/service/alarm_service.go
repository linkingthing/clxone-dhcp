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
		Value:    2000,
		SendMail: true,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalInformPacketWithoutSourceAddrAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalPacketWithOpcodeAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalClientAlarm,
			Level: alarmProto.ThresholdLevel_major,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalClientWithHighQpsAlarm,
			Level: alarmProto.ThresholdLevel_major,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithUnexpectedMessageTypeAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithUltraShortLeaseTimeAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithUltraLongLeaseTimeAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithInvalidServerIdAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithUnexpectedServerIdAlarm,
			Level: alarmProto.ThresholdLevel_major,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithForbiddenServerIdAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithMandatoryServerIdAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithInvalidClientIdAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  false,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionWithMandatoryClientIdAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
	{
		BaseThreshold: &alarmProto.BaseThreshold{
			Name:  alarmProto.ThresholdName_dhcpIllegalOptionsAlarm,
			Level: alarmProto.ThresholdLevel_minor,
			Type:  alarmProto.ThresholdType_trigger,
		},
		Value:    0,
		SendMail: false,
		Enabled:  true,
	},
}

var globalAlarm *alarm.Alarm

func GetAlarmService() *alarm.Alarm {
	return globalAlarm
}

func NewAlarmService(conf *config.DHCPConfig) (err error) {
	globalAlarm, err = alarm.RegisterAlarm(alarm.KafkaConf{
		Addresses: conf.Kafka.Addrs,
		Username:  conf.Kafka.Username,
		Password:  conf.Kafka.Password,
		Topic:     alarm.ThresholdDhcpTopic,
		GroupId:   conf.Kafka.GroupUpdateThresholdEvent,
	}, baseThreshold...)
	return
}
