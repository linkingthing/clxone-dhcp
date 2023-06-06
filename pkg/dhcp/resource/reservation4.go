package resource

import (
	"fmt"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	ActionBatchDelete          = "batch_delete"
	ActionListToReservation    = "list_to_reservation"
	ActionDynamicToReservation = "dynamic_to_reservation"
)

var TableReservation4 = restdb.ResourceDBType(&Reservation4{})

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

type ConvToReservationInput struct {
	Addresses       []string        `json:"addresses"`
	ReservationType ReservationType `json:"reservationType"`
	BothV4V6        bool            `json:"bothV4V6"`
}

type ConvToReservationItem struct {
	Address    string   `json:"address"`
	DualStacks []string `json:"dualStacks"`
	HwAddress  string   `json:"hwAddress"`
	Hostname   string   `json:"hostname"`
	Duid       string   `json:"duid"`
}

type ConvToReservationOutput struct {
	Data []ConvToReservationItem `json:"data"`
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
		return fmt.Errorf("hwaddress and hostname must have only one")
	}

	if r.HwAddress != "" {
		if hw, err := util.NormalizeMac(r.HwAddress); err != nil {
			return fmt.Errorf("hwaddress %s is invalid", r.HwAddress)
		} else {
			r.HwAddress = hw
		}
	} else if r.Hostname != "" {
		if err := util.ValidateStrings(util.RegexpTypeCommon, r.Hostname); err != nil {
			return err
		}
	}

	if ipv4, err := gohelperip.ParseIPv4(r.IpAddress); err != nil {
		return err
	} else {
		r.Ip = ipv4
	}

	if err := util.ValidateStrings(util.RegexpTypeComma, r.Comment); err != nil {
		return err
	}

	r.Capacity = 1
	return nil
}
