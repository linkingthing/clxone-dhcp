package resource

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/linkingthing/cement/set"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
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
	if len(s.Name) == 0 || util.ValidateStrings(s.Name) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, s.Name)
	}

	if len(s.SubnetIds) <= 1 {
		return errorno.ErrSharedNetSubnetIds(s.Name)
	}

	if err := util.ValidateStrings(s.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, s.Comment)
	}

	var sharedNetworks []*SharedNetwork4
	var subnet4s []*Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.FillEx(&subnet4s, genGetSubnetsSqlWithSubnetIds(s.SubnetIds)); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		}

		if err := tx.FillEx(&sharedNetworks, "select * from gr_shared_network4 where id != $1", s.GetID()); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameSharedNetwork), pg.Error(err).Error())
		}

		return nil
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
		return errorno.ErrExpect(errorno.ErrNameSharedNetwork,
			getSubnetIds(subnet4s), s.SubnetIds)
	}

	nodeSet := getIntersectionNodes(subnet4s[0].Nodes, subnet4s[1].Nodes)
	if nodeSet == nil {
		return errorno.ErrNoIntersectionNodes(subnet4s[0].Subnet, subnet4s[1].Subnet)
	}

	var subnets []string
	for _, subnet4 := range subnet4s {
		if len(subnet4.Nodes) == 0 {
			return errorno.ErrNoNode(errorno.ErrNameNetworkV4, subnet4.Subnet)
		} else if !isFullyContains(subnet4.Nodes, nodeSet) {
			return errorno.ErrNotContainNode(errorno.ErrNameNetworkV4,
				subnet4.Subnet, nodeSet.ToSlice())
		} else {
			subnets = append(subnets, subnet4.Subnet)
		}
	}

	s.Subnets = subnets
	return nil
}

func getSubnetIds(subnets []*Subnet4) []string {
	ids := make([]string, 0, len(subnets))
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
			return errorno.ErrConflict(errorno.ErrNameSharedNetwork, errorno.ErrNameSharedNetwork,
				s.Name, name)
		}
	}

	return nil
}
