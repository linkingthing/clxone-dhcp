package resource

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableStaticAddress = restdb.ResourceDBType(&StaticAddress{})

type StaticAddress struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string         `json:"-" db:"ownby"`
	HwAddress                 string         `json:"hwAddress" rest:"required=true" db:"uk"`
	IpAddress                 string         `json:"ipAddress" rest:"required=true" db:"uk"`
	Capacity                  uint64         `json:"capacity" rest:"description=readonly"`
	Version                   util.IPVersion `json:"version" rest:"description=readonly"`
}

func (s StaticAddress) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}

func (s *StaticAddress) String() string {
	return s.HwAddress + "-" + s.IpAddress
}

type StaticAddresses []*StaticAddress

func (s StaticAddresses) Len() int {
	return len(s)
}

func (s StaticAddresses) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s StaticAddresses) Less(i, j int) bool {
	return util.OneIpLessThanAnother(s[i].IpAddress, s[j].IpAddress)
}

func (s *StaticAddress) Validate() error {
	if version, err := checkMacAndIp(s.HwAddress, s.IpAddress); err != nil {
		return fmt.Errorf("static address is invalid %s", err.Error())
	} else {
		s.Version = version
	}

	return nil
}
