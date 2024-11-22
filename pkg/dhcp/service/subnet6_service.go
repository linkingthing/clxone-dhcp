package service

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	Subnet6FileNamePrefix       = "subnet6-"
	Subnet6TemplateFileName     = "subnet6-template"
	Subnet6ImportFileNamePrefix = "subnet6-import"
)

type Subnet6Service struct{}

func NewSubnet6Service() *Subnet6Service {
	return &Subnet6Service{}
}

func (s *Subnet6Service) Create(subnet *resource.Subnet6) error {
	if err := subnet.Validate(nil, nil, nil); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkSubnet6CouldBeCreated(tx, subnet.Subnet); err != nil {
			return err
		}

		if err := setSubnet6ID(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameNetwork, subnet.Subnet, err)
		}

		return sendCreateSubnet6CmdToDHCPAgent(subnet)
	})
}

func checkSubnet6CouldBeCreated(tx restdb.Transaction, subnet string) error {
	if count, err := tx.Count(resource.TableSubnet6, nil); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	} else if count >= int64(config.GetMaxSubnetsCount()) {
		return errorno.ErrExceedMaxCount(errorno.ErrNameNetworkV6, config.GetMaxSubnetsCount())
	}

	var subnets []*resource.Subnet6
	if err := tx.FillEx(&subnets,
		"SELECT * FROM gr_subnet6 WHERE $1 && ipnet", subnet); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	} else if len(subnets) != 0 {
		return errorno.ErrExistIntersection(subnet, subnets[0].Subnet)
	}

	return nil
}

func setSubnet6ID(tx restdb.Transaction, subnet *resource.Subnet6) error {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{
		resource.SqlOrderBy: "subnet_id desc", resource.SqlOffset: 0, resource.SqlLimit: 1},
		&subnets); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if len(subnets) != 0 {
		subnet.SubnetId = subnets[0].SubnetId + 1
	} else {
		subnet.SubnetId = 1
	}

	subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
	return nil
}

func sendCreateSubnet6CmdToDHCPAgent(subnet *resource.Subnet6) error {
	return kafka.SendDHCPCmdWithNodes(false, subnet.Nodes, kafka.CreateSubnet6,
		subnet6ToCreateSubnet6Request(subnet), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteSubnet6,
				&pbdhcpagent.DeleteSubnet6Request{Id: subnet.SubnetId}); err != nil {
				log.Errorf("create subnet6 %s failed, and rollback with nodes %v failed: %s",
					subnet.Subnet, nodesForSucceed, err.Error())
			}
		})
}

func subnet6ToCreateSubnet6Request(subnet *resource.Subnet6) *pbdhcpagent.CreateSubnet6Request {
	return &pbdhcpagent.CreateSubnet6Request{
		Id:                       subnet.SubnetId,
		Subnet:                   subnet.Subnet,
		ValidLifetime:            subnet.ValidLifetime,
		MaxValidLifetime:         subnet.MaxValidLifetime,
		MinValidLifetime:         subnet.MinValidLifetime,
		PreferredLifetime:        subnet.PreferredLifetime,
		MinPreferredLifetime:     subnet.PreferredLifetime,
		MaxPreferredLifetime:     subnet.PreferredLifetime,
		RenewTime:                subnet.PreferredLifetime / 2,
		RebindTime:               subnet.PreferredLifetime * 4 / 5,
		WhiteClientClassStrategy: subnet.WhiteClientClassStrategy,
		WhiteClientClasses:       subnet.WhiteClientClasses,
		BlackClientClassStrategy: subnet.BlackClientClassStrategy,
		BlackClientClasses:       subnet.BlackClientClasses,
		IfaceName:                subnet.IfaceName,
		RelayAgentAddresses:      subnet.RelayAgentAddresses,
		RelayAgentInterfaceId:    subnet.RelayAgentInterfaceId,
		RapidCommit:              subnet.RapidCommit,
		EmbedIpv4:                subnet.EmbedIpv4,
		UseEui64:                 subnet.UseEui64,
		AddressCode:              subnet.AddressCodeName,
		SubnetOptions:            pbSubnetOptionsFromSubnet6(subnet),
	}
}

func pbSubnetOptionsFromSubnet6(subnet *resource.Subnet6) []*pbdhcpagent.SubnetOption {
	var subnetOptions []*pbdhcpagent.SubnetOption
	if len(subnet.DomainServers) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "name-servers",
			Code: 23,
			Data: strings.Join(subnet.DomainServers, ","),
		})
	}

	if subnet.InformationRefreshTime != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "information-refresh-time",
			Code: 32,
			Data: uint32ToString(subnet.InformationRefreshTime),
		})
	}

	if len(subnet.CapWapACAddresses) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "cap-wap-access-controller-addresses",
			Code: 52,
			Data: strings.Join(subnet.CapWapACAddresses, ","),
		})
	}

	return subnetOptions
}

func (s *Subnet6Service) List(ctx *restresource.Context) ([]*resource.Subnet6, error) {
	listCtx := genGetSubnetsContext(ctx, resource.TableSubnet6)
	var subnets []*resource.Subnet6
	var addressCodes []*resource.AddressCode
	var subnetsCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableSubnet6, listCtx.countSql,
				listCtx.params[:len(listCtx.params)-2]...); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
			} else {
				subnetsCount = int(count)
			}
		}

		if err := tx.FillEx(&subnets, listCtx.sql, listCtx.params...); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &addressCodes); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAddressCode), pg.Error(err).Error())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if err := SetSubnet6sLeasesUsedInfo(subnets, listCtx.isUseIds()); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	if nodeNames, err := GetAgentInfo(false, kafka.AgentRoleSentry6); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		setSubnet6sNodeNames(subnets, nodeNames)
	}

	setAddressCode(subnets, addressCodes)
	setPagination(ctx, listCtx.hasPagination, subnetsCount)
	return subnets, nil
}

func SetSubnet6sLeasesUsedInfo(subnets []*resource.Subnet6, useIds bool) (err error) {
	if len(subnets) == 0 {
		return nil
	}

	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			if !resource.IsCapacityZero(subnet.Capacity) && len(subnet.Nodes) != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
				resp, err = client.GetSubnets6LeasesCountWithIds(
					ctx, &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
				return err
			})
		} else {
			return
		}
	} else {
		err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			resp, err = client.GetSubnets6LeasesCount(
				ctx, &pbdhcpagent.GetSubnetsLeasesCountRequest{})
			return err
		})
	}

	if err != nil {
		return errorno.ErrNetworkError(errorno.ErrNameUser, err.Error())
	}

	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		setSubnet6LeasesUsedRatio(subnet, subnetsLeasesCount[subnet.SubnetId])
	}

	return
}

func setSubnet6LeasesUsedRatio(subnet *resource.Subnet6, leasesCount uint64) {
	if !resource.IsCapacityZero(subnet.Capacity) && leasesCount != 0 {
		subnet.UsedCount = leasesCount
		subnet.UsedRatio = fmt.Sprintf("%.4f", calculateUsedRatio(subnet.Capacity, leasesCount))
	}
}

func calculateUsedRatio(capacity string, leasesCount uint64) float64 {
	capacityFloat, _ := new(big.Float).SetString(capacity)
	ratio, _ := new(big.Float).Quo(new(big.Float).SetUint64(leasesCount), capacityFloat).Float64()
	return ratio
}

func setSubnet6sNodeNames(subnets []*resource.Subnet6, nodeNames map[string]Agent) {
	for _, subnet := range subnets {
		subnet.NodeNames, subnet.NodeIds = getSubnetNodeNamesAndIds(subnet.Nodes, nodeNames)
	}
}

func setAddressCode(subnets []*resource.Subnet6, addressCodes []*resource.AddressCode) {
	if len(subnets) == 0 || len(addressCodes) == 0 {
		return
	}

	addrCodeNames := make(map[string]string, len(addressCodes))
	for _, addressCode := range addressCodes {
		addrCodeNames[addressCode.GetID()] = addressCode.Name
	}

	for i := range subnets {
		if name, ok := addrCodeNames[subnets[i].AddressCode]; ok {
			subnets[i].AddressCodeName = name
		}
	}
}

func (s *Subnet6Service) Get(id string) (*resource.Subnet6, error) {
	var subnet6 *resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if subnet6, err = getSubnet6FromDB(tx, id); err != nil {
			return
		}

		return setSubnet6AddressCodeName(tx, subnet6)
	}); err != nil {
		return nil, err
	}

	setSubnet6LeasesUsedInfo(subnet6)
	if nodeNames, err := GetAgentInfo(false, kafka.AgentRoleSentry6); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		subnet6.NodeNames, subnet6.NodeIds = getSubnetNodeNamesAndIds(subnet6.Nodes, nodeNames)
	}

	return subnet6, nil
}

func setSubnet6AddressCodeName(tx restdb.Transaction, subnet *resource.Subnet6) error {
	if subnet.AddressCode != "" {
		var addressCodes []*resource.AddressCode
		if err := tx.Fill(map[string]interface{}{restdb.IDField: subnet.AddressCode}, &addressCodes); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, subnet.AddressCode, pg.Error(err).Error())
		} else if len(addressCodes) != 0 {
			subnet.AddressCodeName = addressCodes[0].Name
		}
	}

	return nil
}

func setSubnet6LeasesUsedInfo(subnet *resource.Subnet6) {
	leasesCount, err := getSubnet6LeasesCount(subnet)
	if err != nil {
		log.Warnf("get subnet6 %s leases used ratio failed: %s", subnet.GetID(), err.Error())
	}

	setSubnet6LeasesUsedRatio(subnet, leasesCount)
}

func getSubnet6LeasesCount(subnet *resource.Subnet6) (uint64, error) {
	if resource.IsCapacityZero(subnet.Capacity) || len(subnet.Nodes) == 0 {
		return 0, nil
	}

	var err error
	var resp *pbdhcpagent.GetLeasesCountResponse
	err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet6LeasesCount(ctx,
			&pbdhcpagent.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameUser, err.Error())
		}
		return err
	})

	return resp.GetLeasesCount(), err
}

func (s *Subnet6Service) Update(subnet *resource.Subnet6) error {
	if err := subnet.ValidateParams(nil, nil); err != nil {
		return err
	}

	newSubnet := &resource.Subnet6{
		EmbedIpv4:   subnet.EmbedIpv4,
		UseEui64:    subnet.UseEui64,
		AddressCode: subnet.AddressCode,
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkUpdateAutoGenAddrFactor(tx, subnet, newSubnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnValidLifetime:            subnet.ValidLifetime,
			resource.SqlColumnMaxValidLifetime:         subnet.MaxValidLifetime,
			resource.SqlColumnMinValidLifetime:         subnet.MinValidLifetime,
			resource.SqlColumnPreferredLifetime:        subnet.PreferredLifetime,
			resource.SqlColumnDomainServers:            subnet.DomainServers,
			resource.SqlColumnWhiteClientClassStrategy: subnet.WhiteClientClassStrategy,
			resource.SqlColumnWhiteClientClasses:       subnet.WhiteClientClasses,
			resource.SqlColumnBlackClientClassStrategy: subnet.BlackClientClassStrategy,
			resource.SqlColumnBlackClientClasses:       subnet.BlackClientClasses,
			resource.SqlColumnIfaceName:                subnet.IfaceName,
			resource.SqlColumnRelayAgentAddresses:      subnet.RelayAgentAddresses,
			resource.SqlColumnRelayAgentInterfaceId:    subnet.RelayAgentInterfaceId,
			resource.SqlColumnInformationRefreshTime:   subnet.InformationRefreshTime,
			resource.SqlColumnCapWapACAddresses:        subnet.CapWapACAddresses,
			resource.SqlColumnTags:                     subnet.Tags,
			resource.SqlColumnRapidCommit:              subnet.RapidCommit,
			resource.SqlColumnEmbedIpv4:                subnet.EmbedIpv4,
			resource.SqlColumnUseEui64:                 subnet.UseEui64,
			resource.SqlColumnAddressCode:              subnet.AddressCode,
			resource.SqlColumnCapacity:                 subnet.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, subnet.GetID(), pg.Error(err).Error())
		}

		return sendUpdateSubnet6CmdToDHCPAgent(subnet)
	})
}

func setSubnet6FromDB(tx restdb.Transaction, subnet *resource.Subnet6) error {
	oldSubnet, err := getSubnet6FromDB(tx, subnet.GetID())
	if err != nil {
		return err
	}

	subnet.SubnetId = oldSubnet.SubnetId
	subnet.Capacity = oldSubnet.Capacity
	subnet.Subnet = oldSubnet.Subnet
	subnet.Ipnet = oldSubnet.Ipnet
	subnet.Nodes = oldSubnet.Nodes
	subnet.EmbedIpv4 = oldSubnet.EmbedIpv4
	subnet.UseEui64 = oldSubnet.UseEui64
	subnet.AddressCode = oldSubnet.AddressCode
	return nil
}

func getSubnet6FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, subnetId, pg.Error(err).Error())
	} else if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV6, subnetId)
	}

	return subnets[0], nil
}

func checkUpdateAutoGenAddrFactor(tx restdb.Transaction, subnet, newSubnet *resource.Subnet6) error {
	if newSubnet.CheckAutoGenAddrFactorConflict() {
		return errorno.ErrAutoGenAddrFactorConflict()
	}

	maskSize, _ := subnet.Ipnet.Mask.Size()
	if enabledAddressCode := (newSubnet.UseAddressCode() && !subnet.UseAddressCode()); enabledAddressCode ||
		(newSubnet.UseEui64 && !subnet.UseEui64) || (newSubnet.EmbedIpv4 && !subnet.EmbedIpv4) {
		if enabledAddressCode {
			if maskSize < 64 {
				return errorno.ErrAddressCodeMask()
			}
		} else if maskSize != 64 {
			return errorno.ErrExpect(errorno.ErrNameEUI64, 64, maskSize)
		}

		if exists, err := subnetHasPools(tx, subnet); err != nil {
			return err
		} else if exists {
			return errorno.ErrHasPools()
		}

		if enabledAddressCode {
			subnet.Capacity = new(big.Int).Lsh(big.NewInt(1), 128-uint(maskSize)).String()
		} else {
			subnet.Capacity = resource.MaxUint64String
		}
	}

	if (!newSubnet.UseEui64 && subnet.UseEui64) || (!newSubnet.EmbedIpv4 && subnet.EmbedIpv4) ||
		(!newSubnet.UseAddressCode() && subnet.UseAddressCode()) {
		if err := checkSubnet6HasNoBeenAllocated(subnet); err != nil {
			return err
		}

		subnet.Capacity = "0"
	}

	subnet.EmbedIpv4 = newSubnet.EmbedIpv4
	subnet.UseEui64 = newSubnet.UseEui64
	subnet.AddressCode = newSubnet.AddressCode
	return nil
}

func subnetHasPools(tx restdb.Transaction, subnet *resource.Subnet6) (bool, error) {
	if !subnet.CanNotHasPools() && !resource.IsCapacityZero(subnet.Capacity) {
		return true, nil
	}

	if counts, err := tx.CountEx(resource.TableSubnet6, "select count(*) from gr_pool6 p FULL JOIN gr_reservation6 r on p.subnet6 = r.subnet6 FULL JOIN gr_pd_pool pd on p.subnet6 = pd.subnet6 FULL JOIN gr_reserved_pool6 rp on p.subnet6 = rp.subnet6 FULL JOIN gr_reserved_pd_pool rpd on p.subnet6 = rpd.subnet6 where p.subnet6 = $1 or r.subnet6 = $1 or pd.subnet6 = $1 or rp.subnet6 = $1 or rpd.subnet6 = $1;", subnet.GetID()); err != nil {
		return false, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	} else {
		return counts != 0, nil
	}
}

func sendUpdateSubnet6CmdToDHCPAgent(subnet *resource.Subnet6) error {
	return kafka.SendDHCPCmdWithNodes(false, subnet.Nodes, kafka.UpdateSubnet6,
		&pbdhcpagent.UpdateSubnet6Request{
			Id:                       subnet.SubnetId,
			Subnet:                   subnet.Subnet,
			ValidLifetime:            subnet.ValidLifetime,
			MaxValidLifetime:         subnet.MaxValidLifetime,
			MinValidLifetime:         subnet.MinValidLifetime,
			PreferredLifetime:        subnet.PreferredLifetime,
			MinPreferredLifetime:     subnet.PreferredLifetime,
			MaxPreferredLifetime:     subnet.PreferredLifetime,
			RenewTime:                subnet.PreferredLifetime / 2,
			RebindTime:               subnet.PreferredLifetime * 4 / 5,
			WhiteClientClassStrategy: subnet.WhiteClientClassStrategy,
			WhiteClientClasses:       subnet.WhiteClientClasses,
			BlackClientClassStrategy: subnet.BlackClientClassStrategy,
			BlackClientClasses:       subnet.BlackClientClasses,
			IfaceName:                subnet.IfaceName,
			RelayAgentAddresses:      subnet.RelayAgentAddresses,
			RelayAgentInterfaceId:    subnet.RelayAgentInterfaceId,
			RapidCommit:              subnet.RapidCommit,
			EmbedIpv4:                subnet.EmbedIpv4,
			UseEui64:                 subnet.UseEui64,
			AddressCode:              subnet.AddressCodeName,
			SubnetOptions:            pbSubnetOptionsFromSubnet6(subnet),
		}, nil)
}

func (s *Subnet6Service) Delete(subnet *resource.Subnet6) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnet6HasNoBeenAllocated(subnet); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSubnet6,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, subnet.Subnet, pg.Error(err).Error())
		}

		return sendDeleteSubnet6CmdToDHCPAgent(subnet, subnet.Nodes)
	})
}

func checkSubnet6HasNoBeenAllocated(subnet6 *resource.Subnet6) error {
	if leasesCount, err := getSubnet6LeasesCount(subnet6); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameNetworkV6, subnet6.Subnet)
	} else {
		return nil
	}
}

func sendDeleteSubnet6CmdToDHCPAgent(subnet *resource.Subnet6, nodes []string) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteSubnet6,
		&pbdhcpagent.DeleteSubnet6Request{Id: subnet.SubnetId}, nil)
}

func (s *Subnet6Service) UpdateNodes(subnetID string, subnetNode *resource.SubnetNode) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetID)
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnNodes: subnetNode.Nodes},
			map[string]interface{}{restdb.IDField: subnetID}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, subnetID, pg.Error(err).Error())
		}

		return sendUpdateSubnet6NodesCmdToDHCPAgent(tx, subnet6, subnetNode.Nodes)
	})
}

func (h *Subnet6Service) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	var oldSubnet6s []*resource.Subnet6
	if err := db.GetResources(map[string]interface{}{resource.SqlOrderBy: "subnet_id desc"},
		&oldSubnet6s); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if len(oldSubnet6s) >= config.GetMaxSubnetsCount() {
		return nil, errorno.ErrExceedMaxCount(errorno.ErrNameNetworkV6, config.GetMaxSubnetsCount())
	}

	sentryNodes, serverNodes, sentryVip, err := kafka.GetDHCPNodes(kafka.AgentStack6)
	if err != nil {
		return nil, err
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(Subnet6ImportFileNamePrefix, TableHeaderSubnet6Fail, response)
	validSqls, reqsForSentryCreate, reqsForSentryDelete,
		reqForServerCreate, reqForServerDelete, err := parseSubnet6sFromFile(file.Name, oldSubnet6s,
		sentryNodes, sentryVip, response)
	if err != nil {
		return response, err
	}

	if len(validSqls) == 0 {
		return response, nil
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, validSql := range validSqls {
			if _, err := tx.Exec(validSql); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
			}
		}

		if sentryVip != "" {
			return sendCreateSubnet6sAndPoolsCmdToDHCPAgentWithHA(sentryNodes, reqForServerCreate)
		} else {
			return sendCreateSubnet6sAndPoolsCmdToDHCPAgent(serverNodes, reqsForSentryCreate, reqsForSentryDelete,
				reqForServerCreate, reqForServerDelete)
		}
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func parseSubnet6sFromFile(fileName string, oldSubnets []*resource.Subnet6, sentryNodes []string, sentryVip string, response *excel.ImportResult) ([]string, map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, map[string]*pbdhcpagent.DeleteSubnets6Request, *pbdhcpagent.CreateSubnets6AndPoolsRequest, *pbdhcpagent.DeleteSubnets6Request, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return nil, nil, nil, nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return nil, nil, nil, nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0],
		TableHeaderSubnet6, SubnetMandatoryFields)
	if err != nil {
		return nil, nil, nil, nil, nil, errorno.ErrInvalidTableHeader()
	}

	dhcpConfig, err := resource.GetDhcpConfig(false)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	clientClass6s, err := resource.GetClientClass6s()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	addressCodes, err := resource.GetAddressCodes(nil)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	response.InitData(len(contents) - 1)
	var maxOldSubnetId uint64
	if len(oldSubnets) != 0 {
		maxOldSubnetId = oldSubnets[0].SubnetId
	}

	sentryNodesForCheck := sentryNodes
	if sentryVip != "" {
		sentryNodesForCheck = []string{sentryVip}
	}

	subnets := make([]*resource.Subnet6, 0, len(contents)-1)
	subnetPools := make(map[uint64][]*resource.Pool6)
	subnetReservedPools := make(map[uint64][]*resource.ReservedPool6)
	subnetReservations := make(map[uint64][]*resource.Reservation6)
	subnetPdPools := make(map[uint64][]*resource.PdPool)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, SubnetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen,
				localizationSubnet6ToStrSlice(&resource.Subnet6{}),
				errorno.ErrMissingMandatory(j+2, SubnetMandatoryFields).ErrorCN())
			continue
		}

		subnet, pools, reservedPools, reservations, pdpools, err := parseSubnet6sAndPools(
			tableHeaderFields, fields)
		if err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := subnet.Validate(dhcpConfig, clientClass6s, addressCodes); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkSubnetNodesValid(subnet.Nodes, sentryNodesForCheck); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkSubnet6ConflictWithSubnet6s(subnet, append(oldSubnets, subnets...)); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkReservation6sValid(subnet, reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkReservedPool6sValid(subnet, reservedPools, reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkPool6sValid(subnet, pools, reservedPools, reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkPdPoolsValid(subnet, pdpools, reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet6FailLen, localizationSubnet6ToStrSlice(subnet),
				errorno.TryGetErrorCNMsg(err))
		} else {
			subnet.SubnetId = maxOldSubnetId + uint64(len(subnets)) + 1
			subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
			subnets = append(subnets, subnet)
			if len(pools) != 0 {
				subnetPools[subnet.SubnetId] = pools
			}

			if len(reservedPools) != 0 {
				subnetReservedPools[subnet.SubnetId] = reservedPools
			}

			if len(reservations) != 0 {
				subnetReservations[subnet.SubnetId] = reservations
			}

			if len(pdpools) != 0 {
				subnetPdPools[subnet.SubnetId] = pdpools
			}
		}
	}

	if len(subnets) == 0 {
		return nil, nil, nil, nil, nil, nil
	}

	sqls := make([]string, 0, 5)
	reqsForSentryCreate := make(map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest)
	reqForServerCreate := &pbdhcpagent.CreateSubnets6AndPoolsRequest{}
	reqsForSentryDelete := make(map[string]*pbdhcpagent.DeleteSubnets6Request)
	reqForServerDelete := &pbdhcpagent.DeleteSubnets6Request{}
	subnetAndNodes := make(map[uint64][]string)
	sqls = append(sqls,
		subnet6sToInsertSqlAndRequest(subnets, reqsForSentryCreate, reqForServerCreate,
			reqsForSentryDelete, reqForServerDelete, subnetAndNodes))
	if len(subnetPools) != 0 {
		sqls = append(sqls, pool6sToInsertSqlAndRequest(subnetPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetReservedPools) != 0 {
		sqls = append(sqls, reservedPool6sToInsertSqlAndRequest(subnetReservedPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetReservations) != 0 {
		sqls = append(sqls, reservation6sToInsertSqlAndRequest(subnetReservations,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetPdPools) != 0 {
		sqls = append(sqls, pdpoolsToInsertSqlAndRequest(subnetPdPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	return sqls, reqsForSentryCreate, reqsForSentryDelete, reqForServerCreate, reqForServerDelete, nil
}

func parseSubnet6sAndPools(tableHeaderFields, fields []string) (*resource.Subnet6, []*resource.Pool6, []*resource.ReservedPool6, []*resource.Reservation6, []*resource.PdPool, error) {
	subnet := &resource.Subnet6{}
	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	var err error
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameSubnet:
			subnet.Subnet = strings.TrimSpace(field)
		case FieldNameSubnetName:
			subnet.Tags = strings.TrimSpace(field)
		case FieldNameEmbedIPv4:
			subnet.EmbedIpv4 = internationalizationBoolSwitch(strings.TrimSpace(field))
		case FieldNameEUI64:
			subnet.UseEui64 = internationalizationBoolSwitch(strings.TrimSpace(field))
		case FieldNameAddressCode:
			subnet.AddressCodeName = strings.TrimSpace(field)
		case FieldNameValidLifetime:
			if subnet.ValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools,
					errorno.ErrInvalidParams(errorno.ErrNameLifetime, field)
			}
		case FieldNameMaxValidLifetime:
			if subnet.MaxValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools,
					errorno.ErrInvalidParams(errorno.ErrNameMaxLifetime, field)
			}
		case FieldNameMinValidLifetime:
			if subnet.MinValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools,
					errorno.ErrInvalidParams(errorno.ErrNameMinLifetime, field)
			}
		case FieldNamePreferredLifetime:
			if subnet.PreferredLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools,
					errorno.ErrInvalidParams(errorno.ErrNamePreferLifetime, field)
			}
		case FieldNameDomainServers:
			subnet.DomainServers = splitFieldWithoutSpace(field)
		case FieldNameIfaceName:
			subnet.IfaceName = strings.TrimSpace(field)
		case FieldNameRelayAddresses:
			subnet.RelayAgentAddresses = splitFieldWithoutSpace(field)
		case FieldNameWhiteClientClassStrategy:
			subnet.WhiteClientClassStrategy = internationalizationClientClassStrategy(strings.TrimSpace(field))
		case FieldNameWhiteClientClasses:
			subnet.WhiteClientClasses = splitFieldWithoutSpace(field)
		case FieldNameBlackClientClassStrategy:
			subnet.BlackClientClassStrategy = internationalizationClientClassStrategy(strings.TrimSpace(field))
		case FieldNameBlackClientClasses:
			subnet.BlackClientClasses = splitFieldWithoutSpace(field)
		case FieldNameOption18:
			subnet.RelayAgentInterfaceId = strings.TrimSpace(field)
		case FieldNameOption32:
			if subnet.InformationRefreshTime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools,
					errorno.ErrInvalidParams(FieldNameOption32, field)
			}
		case FieldNameOption52:
			subnet.CapWapACAddresses = splitFieldWithoutSpace(field)
		case FieldNameNodes:
			subnet.Nodes = splitFieldWithoutSpace(field)
		case FieldNamePools:
			if pools, err = parsePool6sFromString(strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools, err
			}
		case FieldNameReservedPools:
			if reservedPools, err = parseReservedPool6sFromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools, err
			}
		case FieldNameReservations:
			if reservations, err = parseReservation6sFromString(strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools, err
			}
		case FieldNamePdPools:
			if pdpools, err = parsePdPoolsFromString(strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, pdpools, err
			}
		}
	}

	return subnet, pools, reservedPools, reservations, pdpools, nil
}

func parsePool6sFromString(field string) ([]*resource.Pool6, error) {
	var pools []*resource.Pool6
	for _, poolStr := range strings.Split(field, resource.CommonDelimiter) {
		poolStr = strings.TrimSpace(poolStr)
		if poolSlices := strings.SplitN(poolStr, resource.PoolDelimiter, 3); len(poolSlices) != 3 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameDhcpPool, poolStr)
		} else {
			pools = append(pools, &resource.Pool6{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
				Comment:      poolSlices[2],
			})
		}
	}

	return pools, nil
}

func parseReservedPool6sFromString(field string) ([]*resource.ReservedPool6, error) {
	var pools []*resource.ReservedPool6
	for _, poolStr := range strings.Split(field, resource.CommonDelimiter) {
		poolStr = strings.TrimSpace(poolStr)
		if poolSlices := strings.SplitN(poolStr, resource.PoolDelimiter, 3); len(poolSlices) != 3 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameDhcpReservedPool, poolStr)
		} else {
			pools = append(pools, &resource.ReservedPool6{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
				Comment:      poolSlices[2],
			})
		}
	}

	return pools, nil
}

func parseReservation6sFromString(field string) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	for _, reservationStr := range strings.Split(field, resource.CommonDelimiter) {
		reservationStr = strings.TrimSpace(reservationStr)
		if reservationSlices := strings.SplitN(reservationStr,
			resource.ReservationDelimiter, 5); len(reservationSlices) != 5 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameDhcpReservation, reservationStr)
		} else {
			reservation := &resource.Reservation6{
				Comment: reservationSlices[4],
			}

			switch reservationSlices[0] {
			case resource.ReservationIdDUID:
				reservation.Duid = reservationSlices[1]
			case resource.ReservationIdMAC:
				reservation.HwAddress = reservationSlices[1]
			case resource.ReservationIdHostname:
				reservation.Hostname = reservationSlices[1]
			default:
				return nil, errorno.ErrOnlySupport(errorno.ErrName(reservationSlices[0]),
					[]string{resource.ReservationIdDUID, resource.ReservationIdMAC, resource.ReservationIdHostname})
			}

			switch reservationSlices[2] {
			case resource.ReservationTypeIps:
				reservation.IpAddresses = strings.Split(reservationSlices[3], resource.ReservationAddrDelimiter)
			case resource.ReservationTypePrefixes:
				reservation.Prefixes = strings.Split(reservationSlices[3], resource.ReservationAddrDelimiter)
			default:
				return nil, errorno.ErrOnlySupport(errorno.ErrName(reservationSlices[2]),
					[]string{resource.ReservationTypeIps, resource.ReservationTypePrefixes})
			}

			reservations = append(reservations, reservation)
		}
	}

	return reservations, nil
}

func parsePdPoolsFromString(field string) ([]*resource.PdPool, error) {
	var pdpools []*resource.PdPool
	for _, pdpoolStr := range strings.Split(field, resource.CommonDelimiter) {
		pdpoolStr = strings.TrimSpace(pdpoolStr)
		if pdpoolSlices := strings.SplitN(pdpoolStr, resource.PoolDelimiter, 4); len(pdpoolSlices) != 4 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNamePdPool, pdpoolStr)
		} else {
			prefixLen, err := strconv.Atoi(pdpoolSlices[1])
			if err != nil {
				return nil, errorno.ErrInvalidParams(errorno.ErrNamePrefix, pdpoolSlices[1])
			}

			delegatedLen, err := strconv.Atoi(pdpoolSlices[2])
			if err != nil {
				return nil, errorno.ErrInvalidParams(errorno.ErrNameDelegatedLen, pdpoolSlices[2])
			}

			pdpools = append(pdpools, &resource.PdPool{
				Prefix:       pdpoolSlices[0],
				PrefixLen:    uint32(prefixLen),
				DelegatedLen: uint32(delegatedLen),
				Comment:      pdpoolSlices[3],
			})
		}
	}

	return pdpools, nil
}

func checkSubnet6ConflictWithSubnet6s(subnet6 *resource.Subnet6, subnets []*resource.Subnet6) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet6) {
			return errorno.ErrConflict(errorno.ErrNameNetworkV6, errorno.ErrNameNetworkV6,
				subnet6.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func checkReservation6sValid(subnet *resource.Subnet6, reservations []*resource.Reservation6) error {
	if len(reservations) == 0 {
		return nil
	}

	if subnet.CanNotHasPools() {
		return errorno.ErrSubnetCanNotHasPools(subnet.Subnet)
	}

	reservationFieldMap := make(map[string]struct{})
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}

		if err := checkReservation6BelongsToIpnet(subnet.Ipnet, reservation); err != nil {
			return err
		}

		if reservation.Duid != "" {
			if _, ok := reservationFieldMap[reservation.Duid]; ok {
				return errorno.ErrDuplicate(errorno.ErrNameDuid, reservation.Duid)
			} else {
				reservationFieldMap[reservation.Duid] = struct{}{}
			}
		} else if reservation.HwAddress != "" {
			if _, ok := reservationFieldMap[reservation.HwAddress]; ok {
				return errorno.ErrDuplicate(errorno.ErrNameMac, reservation.HwAddress)
			} else {
				reservationFieldMap[reservation.HwAddress] = struct{}{}
			}
		} else if reservation.Hostname != "" {
			if _, ok := reservationFieldMap[reservation.Hostname]; ok {
				return errorno.ErrDuplicate(errorno.ErrNameHostname, reservation.Hostname)
			} else {
				reservationFieldMap[reservation.Hostname] = struct{}{}
			}
		}

		if len(reservation.IpAddresses) != 0 {
			for _, ip := range reservation.IpAddresses {
				if _, ok := reservationFieldMap[ip]; ok {
					return errorno.ErrDuplicate(errorno.ErrNameIp, ip)
				} else {
					reservationFieldMap[ip] = struct{}{}
				}
			}
		} else {
			for _, prefix := range reservation.Prefixes {
				if _, ok := reservationFieldMap[prefix]; ok {
					return errorno.ErrDuplicate(errorno.ErrNamePrefix, prefix)
				} else {
					reservationFieldMap[prefix] = struct{}{}
				}
			}
		}

		subnet.AddCapacityWithString(reservation.Capacity)
	}

	return nil
}

func checkReservedPool6sValid(subnet *resource.Subnet6, reservedPools []*resource.ReservedPool6, reservations []*resource.Reservation6) error {
	reservedPoolsLen := len(reservedPools)
	if reservedPoolsLen == 0 {
		return nil
	}

	if err := checkSubnet6IfCanCreateDynamicPool(subnet); err != nil {
		return err
	}

	for i := 0; i < reservedPoolsLen; i++ {
		if err := reservedPools[i].Validate(); err != nil {
			return err
		}

		if !checkIPsBelongsToIpnet(subnet.Ipnet, reservedPools[i].BeginIp,
			reservedPools[i].EndIp) {
			return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool, errorno.ErrNameNetworkV6,
				reservedPools[i].String(), subnet.Subnet)
		}

		for j := i + 1; j < reservedPoolsLen; j++ {
			if reservedPools[i].CheckConflictWithAnother(reservedPools[j]) {
				return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool, errorno.ErrNameDhcpReservedPool,
					reservedPools[i].String(), reservedPools[j].String())
			}
		}

		if err := checkReservedPool6ConflictWithReservation6s(reservedPools[i],
			reservations); err != nil {
			return err
		}
	}

	return nil
}

func checkPool6sValid(subnet *resource.Subnet6, pools []*resource.Pool6, reservedPools []*resource.ReservedPool6, reservations []*resource.Reservation6) error {
	poolsLen := len(pools)
	if poolsLen == 0 {
		return nil
	}

	if err := checkSubnet6IfCanCreateDynamicPool(subnet); err != nil {
		return err
	}

	for i := 0; i < poolsLen; i++ {
		if err := pools[i].Validate(); err != nil {
			return err
		}

		if !checkIPsBelongsToIpnet(subnet.Ipnet,
			pools[i].BeginIp, pools[i].EndIp) {
			return errorno.ErrNotBelongTo(errorno.ErrNameDhcpPool, errorno.ErrNameNetworkV6,
				pools[i].String(), subnet.Subnet)
		}

		for j := i + 1; j < poolsLen; j++ {
			if pools[i].CheckConflictWithAnother(pools[j]) {
				return errorno.ErrConflict(errorno.ErrNameDhcpPool, errorno.ErrNameDhcpPool,
					pools[i].String(), pools[j].String())
			}
		}

		recalculatePool6CapacityWithReservations(pools[i], reservations)
		recalculatePool6CapacityWithReservedPools(pools[i], reservedPools)
		subnet.AddCapacityWithString(pools[i].Capacity)
	}

	return nil
}

func checkPdPoolsValid(subnet *resource.Subnet6, pdpools []*resource.PdPool, reservations []*resource.Reservation6) error {
	pdpoolsLen := len(pdpools)
	if pdpoolsLen == 0 {
		return nil
	}

	if subnet.CanNotHasPools() {
		return errorno.ErrSubnetCanNotHasPools(subnet.Subnet)
	}

	for i := 0; i < pdpoolsLen; i++ {
		if err := pdpools[i].Validate(); err != nil {
			return err
		}

		if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpools[i].PrefixIpnet,
			pdpools[i].PrefixLen); err != nil {
			return err
		}

		for j := i + 1; j < pdpoolsLen; j++ {
			if pdpools[i].CheckConflictWithAnother(pdpools[j]) {
				return errorno.ErrConflict(errorno.ErrNamePdPool, errorno.ErrNamePdPool,
					pdpools[i].String(), pdpools[j].String())
			}
		}

		recalculatePdPoolCapacityWithReservations(pdpools[i], reservations)
		subnet.AddCapacityWithString(pdpools[i].Capacity)
	}

	return nil
}

func subnet6sToInsertSqlAndRequest(subnets []*resource.Subnet6, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryDelete map[string]*pbdhcpagent.DeleteSubnets6Request, reqForServerDelete *pbdhcpagent.DeleteSubnets6Request, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_subnet6 VALUES ")
	for _, subnet := range subnets {
		buf.WriteString(subnet6ToInsertDBSqlString(subnet))
		if len(subnet.Nodes) == 0 {
			continue
		}

		subnetAndNodes[subnet.SubnetId] = subnet.Nodes
		pbSubnet := subnet6ToCreateSubnet6Request(subnet)
		reqForServerCreate.Subnets = append(reqForServerCreate.Subnets, pbSubnet)
		reqForServerDelete.Ids = append(reqForServerDelete.Ids, subnet.SubnetId)
		for _, node := range subnet.Nodes {
			createReq, ok := reqsForSentryCreate[node]
			deleteReq := reqsForSentryDelete[node]
			if !ok {
				createReq = &pbdhcpagent.CreateSubnets6AndPoolsRequest{}
				deleteReq = &pbdhcpagent.DeleteSubnets6Request{}
			}
			createReq.Subnets = append(createReq.Subnets, pbSubnet)
			deleteReq.Ids = append(deleteReq.Ids, subnet.SubnetId)
			reqsForSentryCreate[node] = createReq
			reqsForSentryDelete[node] = deleteReq
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pool6sToInsertSqlAndRequest(subnetPools map[uint64][]*resource.Pool6, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_pool6 VALUES ")
	for subnetId, pools := range subnetPools {
		for _, pool := range pools {
			buf.WriteString(pool6ToInsertDBSqlString(subnetId, pool))
			pbPool := pool6ToCreatePool6Request(subnetId, pool)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.Pools = append(req.Pools, pbPool)
				}
			}

			if found {
				reqForServerCreate.Pools = append(reqForServerCreate.Pools, pbPool)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservedPool6sToInsertSqlAndRequest(subnetReservedPools map[uint64][]*resource.ReservedPool6, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_reserved_pool6 VALUES ")
	for subnetId, pools := range subnetReservedPools {
		for _, pool := range pools {
			buf.WriteString(reservedPool6ToInsertDBSqlString(subnetId, pool))
			pbReservedPool := reservedPool6ToCreateReservedPool6Request(subnetId, pool)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.ReservedPools = append(req.ReservedPools, pbReservedPool)
				}
			}

			if found {
				reqForServerCreate.ReservedPools = append(reqForServerCreate.ReservedPools, pbReservedPool)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservation6sToInsertSqlAndRequest(subnetReservations map[uint64][]*resource.Reservation6, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_reservation6 VALUES ")
	for subnetId, reservations := range subnetReservations {
		for _, reservation := range reservations {
			buf.WriteString(reservation6ToInsertDBSqlString(subnetId, reservation))
			pbReservation := reservation6ToCreateReservation6Request(subnetId, reservation)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.Reservations = append(req.Reservations, pbReservation)
				}
			}

			if found {
				reqForServerCreate.Reservations = append(reqForServerCreate.Reservations, pbReservation)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pdpoolsToInsertSqlAndRequest(subnetPdPools map[uint64][]*resource.PdPool, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_pd_pool VALUES ")
	for subnetId, pdpools := range subnetPdPools {
		for _, pdpool := range pdpools {
			buf.WriteString(pdpoolToInsertDBSqlString(subnetId, pdpool))
			pbPdPool := pdpoolToCreatePdPoolRequest(subnetId, pdpool)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.PdPools = append(req.PdPools, pbPdPool)
				}
			}

			if found {
				reqForServerCreate.PdPools = append(reqForServerCreate.PdPools, pbPdPool)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func sendCreateSubnet6sAndPoolsCmdToDHCPAgentWithHA(sentryNodes []string, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest) error {
	if len(sentryNodes) == 0 {
		return nil
	}

	_, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		sentryNodes, kafka.CreateSubnet6sAndPools, reqForServerCreate)
	return err
}

func sendCreateSubnet6sAndPoolsCmdToDHCPAgent(serverNodes []string, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryDelete map[string]*pbdhcpagent.DeleteSubnets6Request, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqForServerDelete *pbdhcpagent.DeleteSubnets6Request) error {
	if len(reqsForSentryCreate) == 0 {
		return nil
	}

	succeedSentryNodes := make([]string, 0, len(reqsForSentryCreate))
	for node, req := range reqsForSentryCreate {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateSubnet6sAndPools,
			req); err != nil {
			deleteSentrySubnet6s(reqsForSentryDelete, succeedSentryNodes)
			return err
		}

		succeedSentryNodes = append(succeedSentryNodes, node)
	}

	succeedServerNodes := make([]string, 0, len(serverNodes))
	for _, node := range serverNodes {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateSubnet6sAndPools,
			reqForServerCreate); err != nil {
			deleteSentrySubnet6s(reqsForSentryDelete, succeedSentryNodes)
			deleteServerSubnet6s(reqForServerDelete, succeedServerNodes)
			return err
		}

		succeedServerNodes = append(succeedServerNodes, node)
	}

	return nil
}

func deleteSentrySubnet6s(reqs map[string]*pbdhcpagent.DeleteSubnets6Request, nodes []string) {
	for _, node := range nodes {
		if req, ok := reqs[node]; ok {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				[]string{node}, kafka.DeleteSubnet6s, req); err != nil {
				log.Errorf("delete sentry subnet6s with node %s when rollback failed: %s",
					node, err.Error())
			}
		}
	}
}

func deleteServerSubnet6s(req *pbdhcpagent.DeleteSubnets6Request, nodes []string) {
	for _, node := range nodes {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.DeleteSubnet6s, req); err != nil {
			log.Errorf("delete server subnet6s with node %s when rollback failed: %s",
				node, err.Error())
		}
	}
}

func (s *Subnet6Service) ExportExcel() (*excel.ExportFile, error) {
	var subnet6s []*resource.Subnet6
	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	var addrCodes []*resource.AddressCode
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{resource.SqlOrderBy: resource.SqlColumnSubnetId},
			&subnet6s); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &pools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &reservedPools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &pdpools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNamePdPool), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &addrCodes); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAddressCode), pg.Error(err).Error())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	subnetPools := make(map[string][]string)
	for _, pool := range pools {
		poolSlices := subnetPools[pool.Subnet6]
		poolSlices = append(poolSlices, pool.String()+resource.PoolDelimiter+pool.Comment)
		subnetPools[pool.Subnet6] = poolSlices
	}

	subnetReservedPools := make(map[string][]string)
	for _, reservedPool := range reservedPools {
		reservedPoolSlices := subnetReservedPools[reservedPool.Subnet6]
		reservedPoolSlices = append(reservedPoolSlices, reservedPool.String()+resource.PoolDelimiter+reservedPool.Comment)
		subnetReservedPools[reservedPool.Subnet6] = reservedPoolSlices
	}

	subnetReservations := make(map[string][]string)
	for _, reservation := range reservations {
		reservationSlices := subnetReservations[reservation.Subnet6]
		reservationSlices = append(reservationSlices,
			reservation.String()+resource.ReservationDelimiter+reservation.AddrString()+
				resource.ReservationDelimiter+reservation.Comment)
		subnetReservations[reservation.Subnet6] = reservationSlices
	}

	subnetPdPools := make(map[string][]string)
	for _, pdpool := range pdpools {
		pdpoolSlices := subnetPdPools[pdpool.Subnet6]
		pdpoolSlices = append(pdpoolSlices, pdpool.String()+resource.PoolDelimiter+pdpool.Comment)
		subnetPdPools[pdpool.Subnet6] = pdpoolSlices
	}

	setAddressCode(subnet6s, addrCodes)
	strMatrix := make([][]string, 0, len(subnet6s))
	for _, subnet6 := range subnet6s {
		subnetSlices := localizationSubnet6ToStrSlice(subnet6)
		slices := make([]string, TableHeaderSubnet6Len)
		copy(slices, subnetSlices)
		if poolSlices, ok := subnetPools[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-4] = strings.Join(poolSlices, resource.CommonDelimiter)
		}

		if reservedPools, ok := subnetReservedPools[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-3] = strings.Join(reservedPools, resource.CommonDelimiter)
		}

		if reservations, ok := subnetReservations[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-2] = strings.Join(reservations, resource.CommonDelimiter)
		}

		if pdpools, ok := subnetPdPools[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-1] = strings.Join(pdpools, resource.CommonDelimiter)
		}

		strMatrix = append(strMatrix, slices)
	}

	if filepath, err := excel.WriteExcelFile(Subnet6FileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderSubnet6, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport, string(errorno.ErrNameNetworkV6), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Subnet6Service) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(Subnet6TemplateFileName,
		TableHeaderSubnet6, TemplateSubnet6); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport, string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func sendUpdateSubnet6NodesCmdToDHCPAgent(tx restdb.Transaction, subnet6 *resource.Subnet6, newNodes []string) error {
	if len(subnet6.Nodes) == 0 && len(newNodes) == 0 {
		return nil
	}

	if len(subnet6.Nodes) != 0 && len(newNodes) == 0 {
		if err := checkSubnet6HasNoBeenAllocated(subnet6); err != nil {
			return err
		}
	}

	if len(subnet6.Nodes) != 0 && len(newNodes) != 0 {
		if err := checkSubnetCouldBeUpdateNodes(false); err != nil {
			return err
		}
	}

	nodesForDelete, nodesForCreate, err := getChangedNodes(subnet6.Nodes, newNodes, false)
	if err != nil {
		return err
	}

	if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForDelete, kafka.DeleteSubnet6,
		&pbdhcpagent.DeleteSubnet6Request{Id: subnet6.SubnetId}); err != nil {
		return err
	}

	if len(nodesForCreate) == 0 {
		return nil
	}

	req, cmd, err := genCreateSubnets6AndPoolsRequestWithSubnet6(tx, subnet6)
	if err != nil {
		return err
	}

	if succeedNodes, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForCreate, cmd, req); err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			succeedNodes, kafka.DeleteSubnet6,
			&pbdhcpagent.DeleteSubnet6Request{Id: subnet6.SubnetId}); err != nil {
			log.Errorf("delete subnet6 %s with node %v when rollback failed: %s",
				subnet6.Subnet, succeedNodes, err.Error())
		}
		return err
	}

	return nil
}

func genCreateSubnets6AndPoolsRequestWithSubnet6(tx restdb.Transaction, subnet6 *resource.Subnet6) (proto.Message, kafka.DHCPCmd, error) {
	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	var reservedPdPools []*resource.ReservedPdPool
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&pools); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&reservedPools); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&reservations); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&pdpools); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNamePdPool), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&reservedPdPools); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameReservedPdPool), pg.Error(err).Error())
	}

	if err := setSubnet6AddressCodeName(tx, subnet6); err != nil {
		return nil, "", err
	}

	if len(pools) == 0 && len(reservedPools) == 0 && len(reservations) == 0 &&
		len(pdpools) == 0 && len(reservedPdPools) == 0 {
		return subnet6ToCreateSubnet6Request(subnet6), kafka.CreateSubnet6, nil
	}

	req := &pbdhcpagent.CreateSubnets6AndPoolsRequest{
		Subnets: []*pbdhcpagent.CreateSubnet6Request{subnet6ToCreateSubnet6Request(subnet6)},
	}
	for _, pool := range pools {
		req.Pools = append(req.Pools, pool6ToCreatePool6Request(subnet6.SubnetId, pool))
	}

	for _, pool := range reservedPools {
		req.ReservedPools = append(req.ReservedPools,
			reservedPool6ToCreateReservedPool6Request(subnet6.SubnetId, pool))
	}

	for _, reservation := range reservations {
		req.Reservations = append(req.Reservations,
			reservation6ToCreateReservation6Request(subnet6.SubnetId, reservation))
	}

	for _, pdpool := range pdpools {
		req.PdPools = append(req.PdPools,
			pdpoolToCreatePdPoolRequest(subnet6.SubnetId, pdpool))
	}

	for _, pdpool := range reservedPdPools {
		req.ReservedPdPools = append(req.ReservedPdPools,
			reservedPdPoolToCreateReservedPdPoolRequest(subnet6.SubnetId, pdpool))
	}

	return req, kafka.CreateSubnet6sAndPools, nil
}

func (s *Subnet6Service) CouldBeCreated(couldBeCreatedSubnet *resource.CouldBeCreatedSubnet) error {
	if _, err := gohelperip.ParseCIDRv6(couldBeCreatedSubnet.Subnet); err != nil {
		return errorno.ErrParseCIDR(couldBeCreatedSubnet.Subnet)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkSubnet6CouldBeCreated(tx, couldBeCreatedSubnet.Subnet)
	})
}

func (s *Subnet6Service) ListWithSubnets(subnetListInput *resource.SubnetListInput) (*resource.Subnet6ListOutput, error) {
	for _, subnet := range subnetListInput.Subnets {
		if _, err := gohelperip.ParseCIDRv6(subnet); err != nil {
			return nil, errorno.ErrParseCIDR(subnet)
		}
	}

	subnets, err := ListSubnet6sByPrefixes(subnetListInput.Subnets)
	if err != nil {
		return nil, err
	}

	return &resource.Subnet6ListOutput{Subnet6s: subnets}, nil
}

func ListSubnet6sByPrefixes(prefixes []string) ([]*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "SELECT * FROM gr_subnet6 WHERE subnet = ANY ($1::TEXT[])", prefixes)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if err := SetSubnet6sLeasesUsedInfo(subnets, true); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	return subnets, nil
}

func GetSubnet6ByIP(ip string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "SELECT * FROM gr_subnet6 WHERE ipnet >>= $1", ip)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV6, ip)
	} else {
		return subnets[0], nil
	}
}

func GetSubnet6ByPrefix(prefix string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "SELECT * FROM gr_subnet6 WHERE subnet = $1", prefix)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV6, prefix)
	} else {
		return subnets[0], nil
	}
}
