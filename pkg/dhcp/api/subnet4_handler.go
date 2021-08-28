package api

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	MaxSubnetsCount = 10000

	Subnet4FileNamePrefix   = "subnet4-"
	Subnet4TemplateFileName = "subnet4-template"
)

type Subnet4Handler struct {
}

func NewSubnet4Handler() *Subnet4Handler {
	return &Subnet4Handler{}
}

func (s *Subnet4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var subnets []*resource.Subnet4
		if err := tx.Fill(map[string]interface{}{"orderby": "subnet_id desc"}, &subnets); err != nil {
			return fmt.Errorf("get subnets from db failed: %s\n", err.Error())
		}

		if len(subnets) >= MaxSubnetsCount {
			return fmt.Errorf("subnet4s count has reached maximum (1w)")
		}

		subnet.SubnetId = 1
		if len(subnets) > 0 {
			subnet.SubnetId = subnets[0].SubnetId + 1
		}

		subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
		if err := checkSubnet4ConflictWithSubnet4s(subnet, subnets); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Subnet, err.Error()))
	}

	return subnet, nil
}

func checkSubnet4ConflictWithSubnet4s(subnet4 *resource.Subnet4, subnets []*resource.Subnet4) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet4) {
			return fmt.Errorf("subnet4 %s conflict with subnet4 %s", subnet4.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func sendCreateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateSubnet4,
		subnet4ToPbCreateSubnet4Request(subnet))
}

func subnet4ToPbCreateSubnet4Request(subnet *resource.Subnet4) *dhcpagent.CreateSubnet4Request {
	return &dhcpagent.CreateSubnet4Request{
		Id:                  subnet.SubnetId,
		Subnet:              subnet.Subnet,
		ValidLifetime:       subnet.ValidLifetime,
		MaxValidLifetime:    subnet.MaxValidLifetime,
		MinValidLifetime:    subnet.MinValidLifetime,
		RenewTime:           subnet.ValidLifetime / 2,
		RebindTime:          subnet.ValidLifetime * 3 / 4,
		SubnetMask:          subnet.SubnetMask,
		DomainServers:       subnet.DomainServers,
		Routers:             subnet.Routers,
		ClientClass:         subnet.ClientClass,
		IfaceName:           subnet.IfaceName,
		RelayAgentAddresses: subnet.RelayAgentAddresses,
		NextServer:          subnet.NextServer,
		TftpServer:          subnet.TftpServer,
		Bootfile:            subnet.Bootfile,
	}
}

func (s *Subnet4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions, filterSubnet, hasPagination := genGetSubnetsConditions(ctx)
	var subnets []*resource.Subnet4
	var subnetsCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(conditions, &subnets); err != nil {
			return err
		}

		if hasPagination {
			if count, err := tx.Count(resource.TableSubnet4, nil); err != nil {
				return err
			} else {
				subnetsCount = int(count)
			}
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnet4s from db failed: %s", err.Error()))
	}

	subnetsLeasesCount, err := getSubnet4sLeasesCount(subnets, filterSubnet || hasPagination)
	if err != nil {
		log.Warnf("get subnets leases count failed: %s", err.Error())
	}

	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	setPagination(ctx, hasPagination, subnetsCount)
	return subnets, nil
}

func genGetSubnetsConditions(ctx *restresource.Context) (map[string]interface{}, bool, bool) {
	conditions := map[string]interface{}{"orderby": "subnet_id"}
	filterSubnet := false
	hasPagination := false
	if subnet, ok := util.GetFilterValueWithEqModifierFromFilters(util.FileNameSubnet, ctx.GetFilters()); ok {
		filterSubnet = true
		conditions[util.FileNameSubnet] = subnet
	} else {
		pagination := ctx.GetPagination()
		if pagination.PageSize > 0 && pagination.PageNum > 0 {
			hasPagination = true
			conditions["offset"] = (pagination.PageNum - 1) * pagination.PageSize
			conditions["limit"] = pagination.PageSize
		}
	}

	return conditions, filterSubnet, hasPagination
}

func setPagination(ctx *restresource.Context, hasPagination bool, subnetsCount int) {
	if hasPagination {
		pagination := ctx.GetPagination()
		pagination.Total = subnetsCount
		pagination.PageTotal = int(math.Ceil(float64(subnetsCount) / float64(pagination.PageSize)))
		ctx.SetPagination(pagination)
	}
}

func getSubnet4sLeasesCount(subnets []*resource.Subnet4, useIds bool) (map[uint64]uint64, error) {
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			ids = append(ids, subnet.SubnetId)
		}

		resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCountWithIds(context.TODO(),
			&dhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		return resp.GetSubnetsLeasesCount(), err
	} else {
		resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCount(context.TODO(),
			&dhcpagent.GetSubnetsLeasesCountRequest{})
		return resp.GetSubnetsLeasesCount(), err
	}
}

func (s *Subnet4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	var subnets []*resource.Subnet4
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}

	subnet := subnetInterface.(*resource.Subnet4)
	if err := setSubnet4LeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
	}

	return subnet, nil
}

func setSubnet4LeasesUsedRatio(subnet *resource.Subnet4) error {
	leasesCount, err := getSubnet4LeasesCount(subnet)
	if err != nil {
		return err
	}

	if leasesCount != 0 {
		subnet.UsedCount = leasesCount
		subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
	}
	return nil
}

func getSubnet4LeasesCount(subnet *resource.Subnet4) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4LeasesCount(context.TODO(),
		&dhcpagent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	return resp.GetLeasesCount(), err
}

func (s *Subnet4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"valid_lifetime":        subnet.ValidLifetime,
			"max_valid_lifetime":    subnet.MaxValidLifetime,
			"min_valid_lifetime":    subnet.MinValidLifetime,
			"subnet_mask":           subnet.SubnetMask,
			"domain_servers":        subnet.DomainServers,
			"routers":               subnet.Routers,
			"client_class":          subnet.ClientClass,
			"iface_name":            subnet.IfaceName,
			"relay_agent_addresses": subnet.RelayAgentAddresses,
			"next_server":           subnet.NextServer,
			"tftp_server":           subnet.TftpServer,
			"bootfile":              subnet.Bootfile,
			"tags":                  subnet.Tags,
			"network_type":          subnet.NetworkType,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return subnet, nil
}

func setSubnet4FromDB(tx restdb.Transaction, subnet *resource.Subnet4) error {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnet.GetID()}, &subnets); err != nil {
		return fmt.Errorf("get subnet %s from db failed: %s", subnet.GetID(), err.Error())
	}

	if len(subnets) == 0 {
		return fmt.Errorf("no found subnet %s", subnet.GetID())
	}

	subnet.SubnetId = subnets[0].SubnetId
	subnet.Capacity = subnets[0].Capacity
	subnet.Subnet = subnets[0].Subnet
	subnet.Ipnet = subnets[0].Ipnet
	return nil
}

func sendUpdateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateSubnet4,
		&dhcpagent.UpdateSubnet4Request{
			Id:                  subnet.SubnetId,
			Subnet:              subnet.Subnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			RenewTime:           subnet.ValidLifetime / 2,
			RebindTime:          subnet.ValidLifetime * 3 / 4,
			SubnetMask:          subnet.SubnetMask,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			IfaceName:           subnet.IfaceName,
			NextServer:          subnet.NextServer,
			TftpServer:          subnet.TftpServer,
			Bootfile:            subnet.Bootfile,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
}

func (s *Subnet4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if leasesCount, err := getSubnet4LeasesCount(subnet); err != nil {
			return fmt.Errorf("get subnet %s leases count failed: %s", subnet.Subnet, err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete subnet with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TableSubnet4,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeleteSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteSubnet4,
		&dhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId})
}

func (h *Subnet4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case util.ActionNameImportCSV:
		return h.importCSV(ctx)
	case util.ActionNameExportCSV:
		return h.exportCSV(ctx)
	case util.ActionNameExportCSVTemplate:
		return h.exportCSVTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *Subnet4Handler) importCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var oldSubnet4s []*resource.Subnet4
	if err := db.GetResources(map[string]interface{}{"orderby": "subnet_id desc"}, &oldSubnet4s); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4s from db failed: %s", err.Error()))
	}

	if len(oldSubnet4s) >= MaxSubnetsCount {
		return nil, resterror.NewAPIError(resterror.ServerError,
			"subnet4s count has reached maximum (1w)")
	}

	file, ok := ctx.Resource.GetAction().Input.(*util.ImportFile)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action importcsv input invalid"))
	}

	validSqls, createSubnets4AndPools, err := parseSubnet4sFromFile(file.Name, oldSubnet4s)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("parse subnet4s from file %s failed: %s", file.Name, err.Error()))
	}

	if len(validSqls) == 0 {
		return nil, nil
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, validSql := range validSqls {
			if _, err := tx.Exec(validSql); err != nil {
				return fmt.Errorf("batch insert subnet4s to db failed: %s", err.Error())
			}
		}

		return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateSubnet4sAndPools,
			createSubnets4AndPools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("import subnet4s from file %s failed: %s", file.Name, err.Error()))
	}

	return nil, nil
}

func parseSubnet4sFromFile(fileName string, oldSubnets []*resource.Subnet4) ([]string, *dhcpagent.CreateSubnets4AndPoolsRequest, error) {
	contents, err := util.ReadCSVFile(fileName)
	if err != nil {
		return nil, nil, err
	}

	if len(contents) < 2 {
		return nil, nil, nil
	}

	tableHeaderFields, err := util.ParseTableHeader(contents[0], TableHeaderSubnet4, SubnetMandatoryFields)
	if err != nil {
		return nil, nil, err
	}

	oldSubnetsLen := len(oldSubnets)
	subnets := make([]*resource.Subnet4, 0)
	subnetPools := make(map[uint64][]*resource.Pool4)
	subnetReservedPools := make(map[uint64][]*resource.ReservedPool4)
	subnetReservations := make(map[uint64][]*resource.Reservation4)
	fieldcontents := contents[1:]
	for _, fieldcontent := range fieldcontents {
		fields, missingMandatory, emptyLine := util.ParseTableFields(fieldcontent,
			tableHeaderFields, SubnetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			log.Warnf("subnet4 missing mandatory fields subnet")
			continue
		}

		subnet, pools, reservedPools, reservations, err := parseSubnet4sAndPools(tableHeaderFields, fields)
		if err != nil {
			log.Warnf("parse subnet4 %s fields failed: %s", subnet.Subnet, err.Error())
		} else if err := subnet.Validate(); err != nil {
			log.Warnf("subnet %s is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkSubnet4ConflictWithSubnet4s(subnet, append(oldSubnets, subnets...)); err != nil {
			log.Warnf(err.Error())
		} else if err := checkReservationsValid(subnet, reservations); err != nil {
			log.Warnf("subnet %s reservations is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkReservedPool4sValid(subnet, reservedPools, reservations); err != nil {
			log.Warnf("subnet %s reserved pool4s is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkPool4sValid(subnet, pools, reservedPools, reservations); err != nil {
			log.Warnf("subnet %s pool4s is invalid: %s", subnet.Subnet, err.Error())
		} else {
			subnet.SubnetId = uint64(oldSubnetsLen + len(subnets) + 1)
			subnet.SetID(strconv.Itoa(int(subnet.SubnetId)))
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
		}
	}

	if len(subnets) == 0 {
		return nil, nil, nil
	}

	var sqls []string
	req := &dhcpagent.CreateSubnets4AndPoolsRequest{}
	sqls = append(sqls, subnet4sToInsertSqlAndPbRequest(subnets, req))
	if len(subnetPools) != 0 {
		sqls = append(sqls, pool4sToInsertSqlAndPbRequest(subnetPools, req))
	}

	if len(subnetReservedPools) != 0 {
		sqls = append(sqls, reservedPool4sToInsertSqlAndPbRequest(subnetReservedPools, req))
	}

	if len(subnetReservations) != 0 {
		sqls = append(sqls, reservation4sToInsertSqlAndPbRequest(subnetReservations, req))
	}

	return sqls, req, nil
}

func parseUint32FromString(field string) (uint32, error) {
	value, err := strconv.ParseUint(field, 10, 32)
	return uint32(value), err
}

func parseSubnet4sAndPools(tableHeaderFields, fields []string) (*resource.Subnet4, []*resource.Pool4, []*resource.ReservedPool4, []*resource.Reservation4, error) {
	subnet := &resource.Subnet4{}
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	var err error
	for i, field := range fields {
		if util.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameSubnet:
			subnet.Subnet = strings.TrimSpace(field)
		case FieldNameSubnetName:
			subnet.Tags = field
		case FieldNameSubnetType:
			subnet.NetworkType = field
		case FieldNameValidLifetime:
			if subnet.ValidLifetime, err = parseUint32FromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameMaxValidLifetime:
			if subnet.MaxValidLifetime, err = parseUint32FromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameMinValidLifetime:
			if subnet.MinValidLifetime, err = parseUint32FromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameSubnetMask:
			subnet.SubnetMask = strings.TrimSpace(field)
		case FieldNameRouters:
			subnet.Routers = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameDomainServers:
			subnet.DomainServers = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameIfaceName:
			subnet.IfaceName = strings.TrimSpace(field)
		case FieldNameOption60:
			subnet.ClientClass = field
		case FieldNameOption82:
			subnet.RelayAgentAddresses = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameOption66:
			subnet.TftpServer = strings.TrimSpace(field)
		case FieldNameOption67:
			subnet.Bootfile = field
		case FieldNamePools:
			if pools, err = parsePool4sFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameReservedPools:
			if reservedPools, err = parseReservedPool4sFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameReservations:
			if reservations, err = parseReservation4sFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		}
	}

	return subnet, pools, reservedPools, reservations, err
}

func parsePool4sFromString(field string) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	for _, poolStr := range strings.Split(field, ",") {
		if poolSlices := strings.Split(poolStr, "-"); len(poolSlices) != 2 {
			return nil, fmt.Errorf("parse subnet4 pool %s failed with wrong regexp", poolStr)
		} else {
			pools = append(pools, &resource.Pool4{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
			})
		}
	}

	return pools, nil
}

func parseReservedPool4sFromString(field string) ([]*resource.ReservedPool4, error) {
	var pools []*resource.ReservedPool4
	for _, poolStr := range strings.Split(field, ",") {
		if poolSlices := strings.Split(poolStr, "-"); len(poolSlices) != 2 {
			return nil, fmt.Errorf("parse subnet4 reserved pool %s failed with wrong regexp", poolStr)
		} else {
			pools = append(pools, &resource.ReservedPool4{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
			})
		}
	}

	return pools, nil
}

func parseReservation4sFromString(field string) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	for _, reservationStr := range strings.Split(field, ",") {
		if reservationSlices := strings.Split(reservationStr, "-"); len(reservationSlices) != 2 {
			return nil, fmt.Errorf("parse subnet4 reservation %s failed with wrong regexp", reservationStr)
		} else {
			reservations = append(reservations, &resource.Reservation4{
				HwAddress: reservationSlices[0],
				IpAddress: reservationSlices[1],
			})
		}
	}

	return reservations, nil
}

func checkReservationsValid(subnet4 *resource.Subnet4, reservations []*resource.Reservation4) error {
	ipMacs := make(map[string]struct{})
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet4.Ipnet, reservation.IpAddress) == false {
			return fmt.Errorf("reservation %s not belongs to subnet %s",
				reservation.IpAddress, subnet4.Subnet)
		}

		if _, ok := ipMacs[reservation.IpAddress]; ok {
			return fmt.Errorf("duplicate reservation with ip %s", reservation.IpAddress)
		} else if _, ok := ipMacs[reservation.HwAddress]; ok {
			return fmt.Errorf("duplicate reservation with mac %s", reservation.HwAddress)
		} else {
			ipMacs[reservation.IpAddress] = struct{}{}
			ipMacs[reservation.HwAddress] = struct{}{}
		}
	}

	subnet4.Capacity += uint64(len(reservations))
	return nil
}

func checkReservedPool4sValid(subnet4 *resource.Subnet4, reservedPools []*resource.ReservedPool4, reservations []*resource.Reservation4) error {
	reservedPoolsLen := len(reservedPools)
	for i := 0; i < reservedPoolsLen; i++ {
		if err := reservedPools[i].Validate(); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet4.Ipnet, reservedPools[i].BeginAddress,
			reservedPools[i].EndAddress) == false {
			return fmt.Errorf("reserved pool %s not belongs to subnet %s",
				reservedPools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < reservedPoolsLen; j++ {
			if reservedPools[i].CheckConflictWithAnother(reservedPools[j]) {
				return fmt.Errorf("reserved pool %s conflict with another %s",
					reservedPools[i].String(), reservedPools[j].String())
			}
		}

		for _, reservation := range reservations {
			if reservedPools[i].Contains(reservation.IpAddress) {
				return fmt.Errorf("reserved pool %s conflict with reservation %s",
					reservedPools[i].String(), reservation.String())
			}
		}
	}
	return nil
}

func checkPool4sValid(subnet4 *resource.Subnet4, pools []*resource.Pool4, reservedPools []*resource.ReservedPool4, reservations []*resource.Reservation4) error {
	poolsLen := len(pools)
	for i := 0; i < poolsLen; i++ {
		if err := pools[i].Validate(); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet4.Ipnet, pools[i].BeginAddress, pools[i].EndAddress) == false {
			return fmt.Errorf("pool %s not belongs to subnet %s", pools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < poolsLen; j++ {
			if pools[i].CheckConflictWithAnother(pools[j]) {
				return fmt.Errorf("pool %s conflict with another %s", pools[i].String(), pools[j].String())
			}
		}

		for _, reservation := range reservations {
			if pools[i].Contains(reservation.IpAddress) {
				pools[i].Capacity -= reservation.Capacity
			}
		}

		for _, reservedPool := range reservedPools {
			if pools[i].CheckConflictWithReservedPool4(reservedPool) {
				pools[i].Capacity -= getPool4ReservedCountWithReservedPool4(pools[i], reservedPool)
			}
		}

		subnet4.Capacity += pools[i].Capacity
	}
	return nil
}

func subnet4sToInsertSqlAndPbRequest(subnets []*resource.Subnet4, req *dhcpagent.CreateSubnets4AndPoolsRequest) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_subnet4 values ")
	for _, subnet := range subnets {
		buf.WriteString(subnet4ToInsertDBSqlString(subnet))
		req.Subnets = append(req.Subnets, subnet4ToPbCreateSubnet4Request(subnet))
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pool4sToInsertSqlAndPbRequest(subnetPools map[uint64][]*resource.Pool4, req *dhcpagent.CreateSubnets4AndPoolsRequest) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_pool4 values ")
	for subnetId, pools := range subnetPools {
		for _, pool := range pools {
			buf.WriteString(pool4ToInsertDBSqlString(subnetId, pool))
			req.Pools = append(req.Pools, pool4ToPbCreatePool4Request(subnetId, pool))
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservedPool4sToInsertSqlAndPbRequest(subnetReservedPools map[uint64][]*resource.ReservedPool4, req *dhcpagent.CreateSubnets4AndPoolsRequest) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_reserved_pool4 values ")
	for subnetId, pools := range subnetReservedPools {
		for _, pool := range pools {
			buf.WriteString(reservedPool4ToInsertDBSqlString(subnetId, pool))
			req.ReservedPools = append(req.ReservedPools,
				reservedPool4ToPbCreateReservedPool4Request(subnetId, pool))
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservation4sToInsertSqlAndPbRequest(subnetReservations map[uint64][]*resource.Reservation4, req *dhcpagent.CreateSubnets4AndPoolsRequest) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_reservation4 values ")
	for subnetId, reservations := range subnetReservations {
		for _, reservation := range reservations {
			buf.WriteString(reservation4ToInsertDBSqlString(subnetId, reservation))
			req.Reservations = append(req.Reservations,
				reservation4ToPbCreateReservation4Request(subnetId, reservation))
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func (h *Subnet4Handler) exportCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var subnet4s []*resource.Subnet4
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{"orderby": "subnet_id"}, &subnet4s); err != nil {
			return err
		}

		if err := tx.Fill(nil, &pools); err != nil {
			return err
		}

		if err := tx.Fill(nil, &reservedPools); err != nil {
			return err
		}

		return tx.Fill(nil, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet4 failed: %s", err.Error()))
	}

	subnetPools := make(map[string][]string)
	for _, pool := range pools {
		poolSlices := subnetPools[pool.Subnet4]
		poolSlices = append(poolSlices, pool.String())
		subnetPools[pool.Subnet4] = poolSlices
	}

	subnetReservedPools := make(map[string][]string)
	for _, reservedPool := range reservedPools {
		reservedPoolSlices := subnetReservedPools[reservedPool.Subnet4]
		reservedPoolSlices = append(reservedPoolSlices, reservedPool.String())
		subnetReservedPools[reservedPool.Subnet4] = reservedPoolSlices
	}

	subnetReservations := make(map[string][]string)
	for _, reservation := range reservations {
		reservationSlices := subnetReservations[reservation.Subnet4]
		reservationSlices = append(reservationSlices, reservation.String())
		subnetReservations[reservation.Subnet4] = reservationSlices
	}

	var strMatrix [][]string
	for _, subnet4 := range subnet4s {
		slices := localizationSubnet4ToStrSlice(subnet4)
		if poolSlices, ok := subnetPools[subnet4.GetID()]; ok {
			slices = append(slices, strings.Join(poolSlices, ","))
		}

		if reservedPools, ok := subnetReservedPools[subnet4.GetID()]; ok {
			slices = append(slices, strings.Join(reservedPools, ","))
		}

		if reservations, ok := subnetReservations[subnet4.GetID()]; ok {
			slices = append(slices, strings.Join(reservations, ","))
		}

		strMatrix = append(strMatrix, slices)
	}

	if filepath, err := util.WriteCSVFile(Subnet4FileNamePrefix+time.Now().Format(util.TimeFormat),
		TableHeaderSubnet4, strMatrix); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export device failed: %s", err.Error()))
	} else {
		return &util.ExportFile{Path: filepath}, nil
	}
}

func (h *Subnet4Handler) exportCSVTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if filepath, err := util.WriteCSVFile(Subnet4TemplateFileName, TableHeaderSubnet4, nil); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet4 template failed: %s", err.Error()))
	} else {
		return &util.ExportFile{Path: filepath}, nil
	}
}
