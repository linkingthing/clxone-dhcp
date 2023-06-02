package service

import (
	"strings"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	Reservation4FileNamePrefix       = "reservation4-"
	Reservation4TemplateFileName     = "reservation4-template"
	Reservation4ImportFileNamePrefix = "reservation4-import"
	Reservation6FileNamePrefix       = "reservation6-"
	Reservation6TemplateFileName     = "reservation6-template"
	Reservation6ImportFileNamePrefix = "reservation6-import"
)

const (
	FieldNameIpAddress                   = "IP地址*"
	FieldNameReservation4DeviceFlag      = "设备标识(MAC/主机名)"
	FieldNameReservation6DeviceFlag      = "设备标识(MAC/主机名/DUID)"
	FieldNameReservation4DeviceFlagValue = "MAC地址/主机名*"
	FieldNameReservation6DeviceFlagValue = "MAC地址/主机名/DUID*"
	FieldNameComment                     = "备注"
)

const (
	ReservationFlagMac      = "MAC地址"
	ReservationFlagHostName = "主机名"
	ReservationFlagDUID     = "DUID"
)

var (
	TableHeaderReservation4 = []string{
		FieldNameIpAddress, FieldNameReservation4DeviceFlag, FieldNameReservation4DeviceFlagValue, FieldNameComment,
	}

	TableHeaderReservation6 = []string{
		FieldNameIpAddress, FieldNameReservation6DeviceFlag, FieldNameReservation6DeviceFlagValue, FieldNameComment,
	}

	TableHeaderReservation4Fail = append(TableHeaderReservation4, FailReasonLocalization)
	TableHeaderReservation6Fail = append(TableHeaderReservation6, FailReasonLocalization)

	ReservationMandatoryFields     = []string{FieldNameIpAddress, FieldNameReservation4DeviceFlag}
	TableHeaderReservation4FailLen = len(TableHeaderReservation4Fail)
	TableHeaderReservation6FailLen = len(TableHeaderReservation6Fail)

	TemplateReservation4 = [][]string{
		{"2000::1111,2000::1112", "MAC", "00:0c:29:df:20:33", ""},
		{"2000::2111", "主机名", "admin电脑1", ""},
		{"2000::3111", "DUID", "000300015489982161be", ""},
	}

	TemplateReservation6 = [][]string{
		{"2000::1111,2000::1112", "MAC", "00:0c:29:df:20:33", ""},
		{"2000::2111", "主机名", "admin电脑1", ""},
		{"2000::3111", "DUID", "000300015489982161be", ""},
	}
)

func localizationReservation4ToStrSlice(reservation4 *resource.Reservation4) []string {
	deviceFlag, deviceFlagValue := getFlagAndValue(reservation4.HwAddress, reservation4.Hostname, "")
	return []string{
		reservation4.IpAddress,
		deviceFlag,
		deviceFlagValue,
		reservation4.Comment,
	}
}

func getFlagAndValue(hwAddress, hostName, duid string) (string, string) {
	deviceFlag := ReservationFlagMac
	deviceFlagValue := hwAddress
	if hostName != "" {
		deviceFlag = ReservationFlagHostName
		deviceFlagValue = hostName
	} else if duid != "" {
		deviceFlag = ReservationFlagDUID
		deviceFlagValue = duid
	}

	return deviceFlag, deviceFlagValue
}

func localizationReservation6ToStrSlice(reservation6 *resource.Reservation6) []string {
	deviceFlag, deviceFlagValue := getFlagAndValue(reservation6.HwAddress, reservation6.Hostname, reservation6.Duid)
	return []string{
		strings.Join(reservation6.IpAddresses, ","),
		deviceFlag,
		deviceFlagValue,
		reservation6.Comment,
	}
}
