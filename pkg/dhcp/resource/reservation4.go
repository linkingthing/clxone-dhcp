package resource

import (
	"net"
	"time"
	"unicode/utf8"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/uuid"
	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	MaxCommentLength = 64

	ActionBatchDelete          = "batch_delete"
	ActionListToReservation    = "list_to_reservation"
	ActionDynamicToReservation = "dynamic_to_reservation"
)

var TableReservation4 = restdb.ResourceDBType(&Reservation4{})

type Reservation4s []*Reservation4
type Reservation4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	HwAddress                 string `json:"hwAddress"`
	Hostname                  string `json:"hostname"`
	IpAddress                 string `json:"ipAddress" rest:"required=true"`
	Ip                        net.IP `json:"-"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	Comment                   string `json:"comment"`
}

func (r Reservation4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

func (s Reservation4) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:  excel.ActionNameImport,
			Input: &excel.ImportFile{},
		},
		restresource.Action{
			Name:   excel.ActionNameExport,
			Output: &excel.ExportFile{},
		},
		restresource.Action{
			Name:   excel.ActionNameExportTemplate,
			Output: &excel.ExportFile{},
		},
		restresource.Action{
			Name:  ActionBatchDelete,
			Input: &BatchDeleteInput{},
		},
	}
}

type BatchDeleteInput struct {
	Ids []string `json:"ids"`
}

func (r *Reservation4) String() string {
	if r.HwAddress != "" {
		return ReservationIdMAC + ReservationDelimiter + r.HwAddress + ReservationDelimiter + r.IpAddress
	} else {
		return ReservationIdHostname + ReservationDelimiter + r.Hostname + ReservationDelimiter + r.IpAddress
	}
}

func (r *Reservation4) Validate() error {
	if (r.HwAddress != "" && r.Hostname != "") || (r.HwAddress == "" && r.Hostname == "") {
		return errorno.ErrOnlyOne(string(errorno.ErrNameMac), string(errorno.ErrNameHostname))
	}

	if r.HwAddress != "" {
		if hw, err := util.NormalizeMac(r.HwAddress); err != nil {
			return err
		} else {
			r.HwAddress = hw
		}
	} else if r.Hostname != "" {
		if err := util.ValidateStrings(util.RegexpTypeCommon, r.Hostname); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameHostname, r.Hostname)
		}
	}

	if ipv4, err := gohelperip.ParseIPv4(r.IpAddress); err != nil {
		return errorno.ErrInvalidAddress(r.IpAddress)
	} else {
		r.Ip = ipv4
	}

	if err := util.ValidateStrings(util.RegexpTypeComma, r.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, r.Comment)
	} else if utf8.RuneCountInString(r.Comment) > MaxCommentLength {
		return errorno.ErrExceedMaxCount(errorno.ErrNameComment, MaxCommentLength)
	}

	r.Capacity = 1
	return nil
}

func (r *Reservation4) GenCopyValues() []interface{} {
	if r.GetID() == "" {
		r.ID, _ = uuid.Gen()
	}
	return []interface{}{
		r.GetID(),
		time.Now(),
		r.Subnet4,
		r.HwAddress,
		r.Hostname,
		r.IpAddress,
		r.Ip,
		r.Capacity,
		r.Comment,
	}
}

func (rs Reservation4s) GetIds() []string {
	result := make([]string, 0, len(rs))
	for _, r := range rs {
		result = append(result, r.GetID())
	}
	return result
}
