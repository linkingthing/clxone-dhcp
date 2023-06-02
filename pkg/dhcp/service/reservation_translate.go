package service

import (
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
	FieldNameIpAddress        = "IP地址*"
	FieldNameDeviceType       = "设备标识(MAC/主机名)"
	FieldNameReservation4Flag = "MAC地址/主机名*"
	FieldNameReservation6Flag = "MAC地址/主机名/DUID*"
	FieldNameComment          = "备注"
)

var (
	TableHeaderReservation4 = []string{
		FieldNameIpAddress, FieldNameDeviceType,
	}

	TableHeaderReservation6 = []string{
		FieldNameIpAddress, FieldNameDeviceType,
	}

	TableHeaderReservation4Fail = append(TableHeaderReservation4, FailReasonLocalization)
	TableHeaderReservation6Fail = append(TableHeaderReservation6, FailReasonLocalization)

	ReservationMandatoryFields     = []string{FieldNameIpAddress, FieldNameDeviceType}
	TableHeaderReservation4FailLen = len(TableHeaderReservation4Fail)
	TableHeaderReservation6FailLen = len(TableHeaderReservation6Fail)

	TemplateReservation4 = [][]string{[]string{
		"127.0.0.0/8", "template", "14400", "28800", "7200", "127.0.0.0", "127.0.0.1",
		"114.114.114.114\n8.8.8.8", "ens33", "option60\noption61", "option3\noption6", "127.0.0.1",
		"linkingthing", "tftp.bin", "1800", "127.0.0.2\n127.0.0.3",
		"127.0.0.6-127.0.0.100-备注1\n127.0.0.106-127.0.0.200-备注2",
		"127.0.0.1-127.0.0.5-备注3\n127.0.0.200-127.0.0.255-备注4",
		"mac$11:11:11:11:11:11$127.0.0.66$备注5\nhostname$linking$127.0.0.101$备注6",
	}}

	TemplateReservation6 = [][]string{
		[]string{"2001::/32", "template1", "关闭", "关闭", "14400", "28800", "7200", "14400",
			"2400:3200::1\n2400:3200::baba:1", "ens33", "2001::255", "option6\noption16", "option21\noption22",
			"Gi0/0/1", "127.0.0.2\n127.0.0.3", "", "", "",
			"2001:0:2001::-48-64-备注1\n2001:0:2002::-48-64-备注2"},
		[]string{"2002::/64", "template2", "关闭", "关闭", "14400", "28800", "7200", "14400",
			"2400:3200::1", "eno1", "2002::255", "option16-1", "option17-1",
			"Gi0/0/2", "127.0.0.3\n127.0.0.4",
			"2002::6-2002::1f-备注1\n2002::26-2002::3f-备注2",
			"2002::1-2002::5-备注3\n2002::20-2002::25-备注4",
			"duid$0102$ips$2002::11_2002::12$备注5\nmac$33:33:33:33:33:33$ips$2002::32_2002::33$备注6\nhostname$linking$ips$2002::34_2002::35$备注7",
			""},
		[]string{"2003::/64", "template3", "开启", "关闭", "14400", "28800", "7200", "14400",
			"2400:3200::baba:1", "eth0", "2003::255", "option16-2", "option17-2", "Gi0/0/3",
			"127.0.0.4\n127.0.0.5", "", "", "", ""},
		[]string{"2004::/64", "template3", "关闭", "开启", "14400", "28800", "7200", "14400",
			"2400:3200::baba:1", "eth0", "2003::255", "option16-2", "option17-2", "Gi0/0/3",
			"127.0.0.4\n127.0.0.5", "", "", "", ""},
	}
)

func localizationReservation4ToStrSlice(reservation4 *resource.Reservation4) []string {
	return []string{
		reservation4.IpAddress,
		reservation4.HwAddress,
		reservation4.Hostname,
		reservation4.Comment,
	}
}

func localizationReservation6ToStrSlice(reservation6 *resource.Reservation6) []string {
	return []string{
		reservation6.IpAddresses[0],
		reservation6.HwAddress,
		reservation6.Hostname,
		reservation6.Comment,
	}
}
