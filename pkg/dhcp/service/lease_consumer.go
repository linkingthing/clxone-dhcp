package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/alarm"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-server"
)

const (
	LeaseRequestTypeRequest = "Request"
	LeaseRequestTypeDecline = "Decline"
)

func ConsumeLease() {
	go consumeLease4()
	go consumeLease6()
}

func consumeLease4() {
	readerLease4 := alarm.GetKafkaConsumer().GetReaderLease4()
	if readerLease4 == nil {
		log.Warnf("lease4 reader had not been init, can`t comsume lease4")
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
		addOuiWithLease4(lease4)
		autoReservation4IfNeed(string(message.Key), lease4)
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
		if exists, err := tx.Exists(resource.TableDhcpFingerprint,
			map[string]interface{}{resource.SqlColumnFingerprint: fingerprint.Fingerprint}); err != nil {
			return err
		} else if exists {
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

func addOuiWithLease4(lease4 pbdhcp.Lease4) {
	if lease4.GetHwAddress() != "" && lease4.GetHwAddressOrganization() == "" {
		addOuiIfNeed(&resource.DhcpOui{
			Oui:        lease4.GetHwAddress()[:8],
			DataSource: resource.DataSourceAuto,
		})
	}
}

func addOuiIfNeed(oui *resource.DhcpOui) {
	oui.Oui = strings.ToUpper(oui.Oui)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableDhcpOui,
			map[string]interface{}{resource.SqlColumnOui: oui.Oui}); err != nil {
			return err
		} else if exists {
			return nil
		}

		oui.SetID(oui.Oui)
		if _, err := tx.Insert(oui); err != nil {
			return err
		}

		return sendCreateOuiCmdToDHCPAgent(oui)
	}); err != nil {
		log.Warnf("add oui %s failed: %s", oui.Oui, err.Error())
	}
}

func autoReservation4IfNeed(requestType string, lease4 pbdhcp.Lease4) {
	log.Debugf("lease consumer request type %v lease with mac %s hostname %s ip %s allocate mode %v lease state: %v\n",
		requestType, lease4.GetHwAddress(), lease4.GetHostname(), lease4.GetAddress(), lease4.GetAllocateMode(), lease4.GetLeaseState())
	if requestType != LeaseRequestTypeRequest && requestType != LeaseRequestTypeDecline {
		return
	}

	if lease4.GetAllocateMode() == 0 && requestType == LeaseRequestTypeRequest {
		autoCreateReservation4IfNeed(lease4)
		return
	}

	if requestType == LeaseRequestTypeDecline {
		autoDeleteReservation4IfNeed(lease4)
	}
}

func autoCreateReservation4IfNeed(lease4 pbdhcp.Lease4) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, strconv.FormatUint(lease4.GetSubnetId(), 10))
		if err != nil {
			return err
		}

		if subnet4.AutoReservationType == resource.AutoReservationTypeNone ||
			(subnet4.AutoReservationType == resource.AutoReservationTypeMac && len(lease4.GetHwAddress()) == 0) ||
			(subnet4.AutoReservationType == resource.AutoReservationTypeHostname && len(lease4.GetHostname()) == 0) {
			return nil
		}

		reservation4 := &resource.Reservation4{IpAddress: lease4.GetAddress(), AutoCreate: true}
		if subnet4.AutoReservationType == resource.AutoReservationTypeMac {
			reservation4.HwAddress = lease4.GetHwAddress()
		} else {
			reservation4.Hostname = lease4.GetHostname()
		}

		if err := reservation4.Validate(); err != nil {
			return err
		}

		return createReservation4(tx, subnet4, reservation4)
	}); err != nil {
		log.Warnf("auto create reservation4 with mac %s hostname %s ip %s failed: %s",
			lease4.GetHwAddress(), lease4.GetHostname(), lease4.GetAddress(), err.Error())
	}
}

func autoDeleteReservation4IfNeed(lease4 pbdhcp.Lease4) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, strconv.FormatUint(lease4.GetSubnetId(), 10))
		if err != nil {
			return err
		}

		if subnet4.AutoReservationType == resource.AutoReservationTypeNone {
			return nil
		}

		var reservations []*resource.Reservation4
		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnIpAddress:  lease4.GetAddress(),
			resource.SqlColumnSubnet4:    subnet4.GetID(),
			resource.SqlColumnAutoCreate: true,
		}, &reservations); err != nil {
			return err
		} else if len(reservations) == 0 {
			return nil
		}

		return deleteReservation4(tx, subnet4, reservations[0])
	}); err != nil {
		log.Warnf("auto delete reservation4 with subnet %d %s and ip %s failed: %s",
			lease4.GetSubnetId(), lease4.GetSubnet(), lease4.GetAddress(), err.Error())
	}
}

func consumeLease6() {
	readerLease6 := alarm.GetKafkaConsumer().GetReaderLease6()
	if readerLease6 == nil {
		log.Warnf("lease6 reader had not been init, can`t comsume lease6")
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
		addOuiWithLease6(lease6)
		autoReservation6IfNeed(string(message.Key), lease6)
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

func addOuiWithLease6(lease6 pbdhcp.Lease6) {
	if lease6.GetHwAddress() != "" && lease6.GetHwAddressOrganization() == "" {
		addOuiIfNeed(&resource.DhcpOui{
			Oui:        lease6.GetHwAddress()[:8],
			DataSource: resource.DataSourceAuto,
		})
	}
}

func autoReservation6IfNeed(requestType string, lease6 pbdhcp.Lease6) {
	log.Debugf("lease consumer request type %v lease with mac %s duid %s hostname %s ip %s allocate mode %v lease state: %v\n",
		requestType, lease6.GetHwAddress(), lease6.GetDuid(), lease6.GetHostname(), lease6.GetAddress(), lease6.GetAllocateMode(), lease6.GetLeaseState())
	if requestType != LeaseRequestTypeRequest && requestType != LeaseRequestTypeDecline {
		return
	}

	if lease6.GetLeaseType() != "IA_NA" {
		return
	}

	if lease6.GetAllocateMode() == 0 && requestType == LeaseRequestTypeRequest {
		autoCreateReservation6IfNeed(lease6)
		return
	}

	if requestType == LeaseRequestTypeDecline {
		autoDeleteReservation6IfNeed(lease6)
	}
}

func autoCreateReservation6IfNeed(lease6 pbdhcp.Lease6) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, strconv.FormatUint(lease6.GetSubnetId(), 10))
		if err != nil {
			return err
		}

		if subnet6.AutoReservationType == resource.AutoReservationTypeNone ||
			(subnet6.AutoReservationType == resource.AutoReservationTypeMac && len(lease6.GetHwAddress()) == 0) ||
			(subnet6.AutoReservationType == resource.AutoReservationTypeDuid && len(lease6.GetDuid()) == 0) ||
			(subnet6.AutoReservationType == resource.AutoReservationTypeHostname && len(lease6.GetHostname()) == 0) {
			return nil
		}

		reservation6 := &resource.Reservation6{IpAddresses: []string{lease6.GetAddress()}, AutoCreate: true}
		if subnet6.AutoReservationType == resource.AutoReservationTypeMac {
			reservation6.HwAddress = lease6.GetHwAddress()
		} else if subnet6.AutoReservationType == resource.AutoReservationTypeDuid {
			reservation6.Duid = lease6.GetDuid()
		} else {
			reservation6.Hostname = lease6.GetHostname()
		}

		if err := reservation6.Validate(); err != nil {
			return err
		}

		return createReservation6(tx, subnet6, reservation6)
	}); err != nil {
		log.Warnf("auto create reservation6 with mac %s duid %s hostname %s ip %s failed: %s",
			lease6.GetHwAddress(), lease6.GetDuid(), lease6.GetHostname(), lease6.GetAddress(), err.Error())
	}
}

func autoDeleteReservation6IfNeed(lease6 pbdhcp.Lease6) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, strconv.FormatUint(lease6.GetSubnetId(), 10))
		if err != nil {
			return err
		}

		if subnet6.AutoReservationType == resource.AutoReservationTypeNone {
			return nil
		}

		var reservations []*resource.Reservation6
		if err := tx.FillEx(&reservations,
			"SELECT * FROM gr_reservation6 WHERE $1 = ANY(ip_addresses) AND subnet6 = $2 AND auto_create = true",
			lease6.GetAddress(), subnet6.GetID()); err != nil {
			return err
		} else if len(reservations) == 0 {
			return nil
		}

		return deleteReservation6(tx, subnet6, reservations[0])
	}); err != nil {
		log.Warnf("auto delete reservation6 with subnet %d %s ip %s failed: %s",
			lease6.GetSubnetId(), lease6.GetSubnet(), lease6.GetAddress(), err.Error())
	}
}
