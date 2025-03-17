package service

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/alarm"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-server"
)

func ConsumeLease() {
	go consumeLease4()
	go consumeLease6()
}

func consumeLease4() {
	readerLease4 := alarm.GetKafkaConsumer().GetReaderLease4()
	if readerLease4 == nil {
		return
	}

	defer readerLease4.Close()
	for {
		message, err := readerLease4.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read lease4 message from kafka failed: %s", err.Error())
			continue
		}

		var lease4 pbdhcp.Lease4
		if err := proto.Unmarshal(message.Value, &lease4); err != nil {
			log.Warnf("unmarshal lease4 message %s failed: %s", message.Key, err.Error())
			continue
		}

		addFingerprintWithLease4(lease4)
	}
}

func addFingerprintWithLease4(lease4 pbdhcp.Lease4) {
	if lease4.GetFingerprint() != "" {
		addFingerprintIfNeed(&resource.DhcpFingerprint{
			Fingerprint:     lease4.GetFingerprint(),
			VendorId:        lease4.GetVendorId(),
			OperatingSystem: lease4.GetOperatingSystem(),
			ClientType:      lease4.GetClientType(),
			MatchPattern:    resource.MatchPatternEqual,
			DataSource:      resource.DataSourceAuto,
		})
	}
}

func addFingerprintIfNeed(fingerprint *resource.DhcpFingerprint) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var fingerprints []*resource.DhcpFingerprint
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnFingerprint: fingerprint.Fingerprint},
			&fingerprints); err != nil {
			return err
		}

		if len(fingerprints) != 0 {
			return nil
		}

		if _, err := tx.Insert(fingerprint); err != nil {
			return err
		}

		return sendCreateFingerprintCmdToAgent(fingerprint)
	}); err != nil {
		log.Warnf("add fingerprint %s failed: %s", fingerprint.Fingerprint, err.Error())
	}
}

func consumeLease6() {
	readerLease6 := alarm.GetKafkaConsumer().GetReaderLease6()
	if readerLease6 == nil {
		return
	}

	defer readerLease6.Close()
	for {
		message, err := readerLease6.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read lease6 message from kafka failed: %s", err.Error())
			continue
		}

		var lease6 pbdhcp.Lease6
		if err := proto.Unmarshal(message.Value, &lease6); err != nil {
			log.Warnf("unmarshal lease6 message %s failed: %s", message.Key, err.Error())
			continue
		}

		addFingerprintWithLease6(lease6)
	}
}

func addFingerprintWithLease6(lease6 pbdhcp.Lease6) {
	if lease6.GetFingerprint() != "" {
		addFingerprintIfNeed(&resource.DhcpFingerprint{
			Fingerprint:     lease6.GetFingerprint(),
			VendorId:        lease6.GetVendorId(),
			OperatingSystem: lease6.GetOperatingSystem(),
			ClientType:      lease6.GetClientType(),
			MatchPattern:    resource.MatchPatternEqual,
			DataSource:      resource.DataSourceAuto,
		})
	}
}
