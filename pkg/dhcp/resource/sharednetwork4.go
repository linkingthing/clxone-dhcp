package resource

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/linkingthing/cement/set"
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
	if len(s.Name) == 0 {
		return fmt.Errorf("shared network4 name is required")
	}

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
	if len(s.SubnetIds) != len(subnet4s) {
		return fmt.Errorf("get subnet4s %v diff from shared subnet ids %v",
			getSubnetIds(subnet4s), s.SubnetIds)
	}

	nodeSet := getIntersectionNodes(subnet4s[0].Nodes, subnet4s[1].Nodes)
	if nodeSet == nil {
		return fmt.Errorf("subnet %s nodes %v has no intersection with subnet %s nodes %v",
			subnet4s[0].Subnet, subnet4s[0].Nodes, subnet4s[1].Subnet, subnet4s[1].Nodes)
	}

	var subnets []string
	for _, subnet4 := range subnet4s {
		if len(subnet4.Nodes) == 0 {
			return fmt.Errorf("subnet %s no nodes info, can`t used by shared network", subnet4.Subnet)
		} else if isFullyContains(subnet4.Nodes, nodeSet) == false {
			return fmt.Errorf("subnet %s nodes %v should contains nodes %v",
				subnet4.Subnet, subnet4.Nodes, nodeSet.ToSlice())
		} else {
			subnets = append(subnets, subnet4.Subnet)
		}
	}

	s.Subnets = subnets
	return nil
}

func getSubnetIds(subnets []*Subnet4) []string {
	var ids []string
	for _, subnet := range subnets {
		ids = append(ids, subnet.GetID())
	}

	return ids
}

func getIntersectionNodes(node1s, node2s []string) set.StringSet {
	return set.StringSetFromSlice(node1s).Union(set.StringSetFromSlice(node2s))
}

func isFullyContains(nodes []string, nodeSet set.StringSet) bool {
	var intersectionNodes []string
	for _, node := range nodes {
		if nodeSet.Member(node) {
			intersectionNodes = append(intersectionNodes, node)
		}
	}

	return len(intersectionNodes) == len(nodeSet)
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
