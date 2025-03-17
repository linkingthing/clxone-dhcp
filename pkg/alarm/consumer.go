package alarm

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/cuityhj/cement/log"
	"github.com/golang/protobuf/proto"
	utils "github.com/linkingthing/clxone-utils/alarm"
	kg "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/linkingthing/clxone-dhcp/config"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-server"
	tpservice "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

type DHCPTopic string

const (
	DHCPTopicLease4  DHCPTopic = "DHCPTopicLease4"
	DHCPTopicLease6  DHCPTopic = "DHCPTopicLease6"
	DHCPTopicPacket4 DHCPTopic = "DHCPTopicPacket4"
	DHCPTopicPacket6 DHCPTopic = "DHCPTopicPacket6"
)

type KafkaConsumer struct {
	readerLease4  *kg.Reader
	readerLease6  *kg.Reader
	readerPacket4 *kg.Reader
	readerPacket6 *kg.Reader
}

var globalKafkaConsumer *KafkaConsumer

func GetKafkaConsumer() *KafkaConsumer {
	return globalKafkaConsumer
}

func Init(conf *config.DHCPConfig) {
	if conf.Kafka.Username == "" || conf.Kafka.Password == "" || len(conf.Kafka.Addrs) == 0 ||
		conf.Kafka.GroupID == "" {
		return
	}

	globalKafkaConsumer := &KafkaConsumer{}
	globalKafkaConsumer.readerLease4 = initKafkaReader(conf, DHCPTopicLease4)
	globalKafkaConsumer.readerLease6 = initKafkaReader(conf, DHCPTopicLease6)
	globalKafkaConsumer.readerPacket4 = initKafkaReader(conf, DHCPTopicPacket4)
	globalKafkaConsumer.readerPacket6 = initKafkaReader(conf, DHCPTopicPacket6)
	globalKafkaConsumer.run()
}

func initKafkaReader(conf *config.DHCPConfig, topic DHCPTopic) *kg.Reader {
	return kg.NewReader(kg.ReaderConfig{
		Brokers:  conf.Kafka.Addrs,
		Topic:    string(topic),
		GroupID:  conf.Kafka.GroupID,
		MinBytes: 10,
		MaxBytes: 104857600,
		Dialer: &kg.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
			SASLMechanism: plain.Mechanism{
				Username: conf.Kafka.Username,
				Password: conf.Kafka.Password,
			},
			TLS: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
}

func (kc *KafkaConsumer) GetReaderLease4() *kg.Reader {
	if kc == nil {
		return nil
	} else {
		return kc.readerLease4
	}
}

func (kc *KafkaConsumer) GetReaderLease6() *kg.Reader {
	if kc == nil {
		return nil
	} else {
		return kc.readerLease6
	}
}

func (kc *KafkaConsumer) run() {
	go kc.consumePacket4()
	go kc.consumePacket6()
}

func (kc *KafkaConsumer) consumePacket4() {
	if kc.readerPacket4 == nil {
		return
	}

	defer kc.readerPacket4.Close()
	for {
		message, err := kc.readerPacket4.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read packet4 message from kafka failed: %s", err.Error())
			continue
		}

		var packet4 pbdhcp.Packet4
		if err := proto.Unmarshal(message.Value, &packet4); err != nil {
			log.Warnf("unmarshal packet4 message %s failed: %s", message.Key, err.Error())
			continue
		}

		switch string(message.Key) {
		case string(AlarmTypeIllegalPacketWithOpCode):
			err = tpservice.GetAlarmService().AddDhcpIllegalPacketWithOpcodeAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4), packet4.GetRequestOpCode())
		case string(AlarmTypeIllegalClient):
			err = tpservice.GetAlarmService().AddDhcpIllegalClientAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4))
		case string(AlarmTypeIllegalOptions):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionsAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4), packet4.GetIllegalOptions())
		case string(AlarmTypeIllegalClientWithHighQPS):
			err = tpservice.GetAlarmService().AddDhcpIllegalClientWithHighQpsAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4), packet4.GetHighRequestRate())
		case string(AlarmTypeIllegalInformPacketWithoutSourceAddr):
			err = tpservice.GetAlarmService().AddDhcpIllegalInformPacketWithoutSourceAddrAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4))
		case string(AlarmTypeIllegalOptionWithUnexpectedMessageType):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithUnexpectedMessageTypeAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4))
		case string(AlarmTypeIllegalOptionWithUnexpectedServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithUnexpectedServerIdAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4),
				packet4.GetIllegalOptionCode(), packet4.GetIllegalOptionData())
		case string(AlarmTypeIllegalOptionWithForbiddenServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithForbiddenServerIdAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4),
				packet4.GetIllegalOptionCode(), packet4.GetIllegalOptionData())
		case string(AlarmTypeIllegalOptionWithMandatoryServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithMandatoryServerIdAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4), packet4.GetIllegalOptionCode())
		case string(AlarmTypeIllegalOptionWithMandatoryClientId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithMandatoryClientIdAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4), packet4.GetIllegalOptionCode())
		case string(AlarmTypeIllegalOptionWithUltraShortLeaseTime):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithUltraShortLeaseTimeAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4),
				packet4.GetIllegalOptionCode(), packet4.GetIllegalOptionData())
		case string(AlarmTypeIllegalOptionWithUltraLongLeaseTime):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithUltraLongLeaseTimeAlarm(
				pbPacket4ToAlarmDHCPClient4(packet4),
				packet4.GetIllegalOptionCode(), packet4.GetIllegalOptionData())
		}

		if err != nil {
			log.Warnf("send packet4 alarm %s to kafka failed: %s", message.Key, err.Error())
		}
	}
}

func pbPacket4ToAlarmDHCPClient4(packet4 pbdhcp.Packet4) *utils.DHCPClient4 {
	return &utils.DHCPClient4{
		HwAddress:             packet4.GetHwAddress(),
		HwAddressOrganization: packet4.GetHwAddressOrganization(),
		Hostname:              packet4.GetHostname(),
		ClientId:              packet4.GetClientId(),
		Fingerprint:           packet4.GetFingerprint(),
		VendorId:              packet4.GetVendorId(),
		OperatingSystem:       packet4.GetOperatingSystem(),
		ClientType:            packet4.GetClientType(),
		MessageType:           packet4.GetMessageType(),
	}
}

func (kc *KafkaConsumer) consumePacket6() {
	if kc.readerPacket6 == nil {
		return
	}

	defer kc.readerPacket6.Close()
	for {
		message, err := kc.readerPacket6.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read packet6 message from kafka failed: %s", err.Error())
			continue
		}

		var packet6 pbdhcp.Packet6
		if err := proto.Unmarshal(message.Value, &packet6); err != nil {
			log.Warnf("unmarshal packet6 message %s failed: %s", message.Key, err.Error())
			continue
		}

		switch string(message.Key) {
		case string(AlarmTypeIllegalClient):
			err = tpservice.GetAlarmService().AddDhcpIllegalClientAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6))
		case string(AlarmTypeIllegalClientWithHighQPS):
			err = tpservice.GetAlarmService().AddDhcpIllegalClientWithHighQpsAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6), packet6.GetHighRequestRate())
		case string(AlarmTypeIllegalOptionWithUnexpectedMessageType):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithUnexpectedMessageTypeAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6))
		case string(AlarmTypeIllegalOptions):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionsAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6), packet6.GetIllegalOptions())
		case string(AlarmTypeIllegalOptionWithUnexpectedServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithUnexpectedServerIdAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6),
				packet6.GetIllegalOptionCode(), packet6.GetIllegalOptionData())
		case string(AlarmTypeIllegalOptionWithForbiddenServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithForbiddenServerIdAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6),
				packet6.GetIllegalOptionCode(), packet6.GetIllegalOptionData())
		case string(AlarmTypeIllegalOptionWithMandatoryServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithMandatoryServerIdAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6), packet6.GetIllegalOptionCode())
		case string(AlarmTypeIllegalOptionWithInvalidServerId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithInvalidServerIdAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6),
				packet6.GetIllegalOptionCode(), packet6.GetIllegalOptionData())
		case string(AlarmTypeIllegalOptionWithMandatoryClientId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithMandatoryClientIdAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6), packet6.GetIllegalOptionCode())
		case string(AlarmTypeIllegalOptionWithInvalidClientId):
			err = tpservice.GetAlarmService().AddDhcpIllegalOptionWithInvalidClientIdAlarm(
				pbPacket6ToAlarmDHCPClient6(packet6),
				packet6.GetIllegalOptionCode(), packet6.GetIllegalOptionData())
		}

		if err != nil {
			log.Warnf("send packet6 alarm %s to kafka failed: %s", message.Key, err.Error())
		}
	}
}

func pbPacket6ToAlarmDHCPClient6(packet6 pbdhcp.Packet6) *utils.DHCPClient6 {
	return &utils.DHCPClient6{
		Duid:                  packet6.GetDuid(),
		HwAddress:             packet6.GetHwAddress(),
		HwAddressOrganization: packet6.GetHwAddressOrganization(),
		Hostname:              packet6.GetHostname(),
		Fingerprint:           packet6.GetFingerprint(),
		VendorId:              packet6.GetVendorId(),
		OperatingSystem:       packet6.GetOperatingSystem(),
		ClientType:            packet6.GetClientType(),
		MessageType:           packet6.GetMessageType(),
		RequestSourceAddr:     packet6.GetRequestSourceAddr(),
	}
}
