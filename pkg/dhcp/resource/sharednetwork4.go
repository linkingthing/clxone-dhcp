package resource

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
)

var TableSharedNetwork4 = restdb.ResourceDBType(&SharedNetwork4{})

type SharedNetwork4 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string   `json:"name" rest:"required=true" db:"uk"`
	SubnetIds                 []uint64 `json:"subnetIds" rest:"required=true"`
	Subnets                   []string `json:"subnets"`
	Comment                   string   `json:"comment"`
}

func (s *SharedNetwork4) Validate() error {
	if len(s.SubnetIds) <= 1 {
		return fmt.Errorf("shared network4 %s subnet ids length should excceed 1", s.Name)
	}

	var sharedNetworks []*SharedNetwork4
	var subnet4s []*Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.FillEx(&subnet4s, genGetSubnetsSqlWithSubnetIds(s.SubnetIds)); err != nil {
			return err
		}

		return tx.FillEx(&sharedNetworks, "select * from gr_shared_network4 where id != $1", s.GetID())
	}); err != nil {
		return err
	}

	if err := s.setSharedNetworkSubnets(subnet4s); err != nil {
		return err
	}

	return s.checkConflictWithOthers(sharedNetworks)
}

func genGetSubnetsSqlWithSubnetIds(subnetIds []uint64) string {
	var buf bytes.Buffer
	buf.WriteString("select * from gr_subnet4 where subnet_id in (")
	for _, subnetId := range subnetIds {
		buf.WriteString(strconv.FormatUint(subnetId, 10))
		buf.WriteString(",")
	}

	return strings.TrimSuffix(buf.String(), ",") + ")"
}

func (s *SharedNetwork4) setSharedNetworkSubnets(subnet4s []*Subnet4) error {
	subnetIdAndSubnet := make(map[uint64]string)
	for _, subnet4 := range subnet4s {
		if len(subnet4.Nodes) == 0 {
			return fmt.Errorf("subnet %s no nodes info, can`t used by shared network", subnet4.Subnet)
		}

		subnetIdAndSubnet[subnet4.SubnetId] = subnet4.Subnet
	}

	var nonexistIds []uint64
	var subnets []string
	for _, subnetId := range s.SubnetIds {
		if subnet, ok := subnetIdAndSubnet[subnetId]; ok {
			subnets = append(subnets, subnet)
		} else {
			nonexistIds = append(nonexistIds, subnetId)
		}
	}

	if len(nonexistIds) != 0 {
		return fmt.Errorf("shared network4 %s has non-exists subnet ids %v", s.Name, nonexistIds)
	}

	s.Subnets = subnets
	return nil
}

func (s *SharedNetwork4) checkConflictWithOthers(sharedNetworks []*SharedNetwork4) error {
	sharedNetworksIds := make(map[uint64]string)
	for _, sharedNetwork := range sharedNetworks {
		for _, id := range sharedNetwork.SubnetIds {
			sharedNetworksIds[id] = sharedNetwork.Name
		}
	}

	for _, id := range s.SubnetIds {
		if name, ok := sharedNetworksIds[id]; ok {
			return fmt.Errorf("shared network %s subnet id %d exists in other shared network %s",
				s.Name, id, name)
		}
	}

	return nil
}
