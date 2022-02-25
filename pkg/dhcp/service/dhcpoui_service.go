package service

import (
	"fmt"
	"strings"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	FieldOUI = "oui"
)

type DhcpOuiService struct{}

func NewDhcpOuiService() *DhcpOuiService {
	return &DhcpOuiService{}
}

func (d *DhcpOuiService) Create(dhcpoui *resource.DhcpOui) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(dhcpoui); err != nil {
			return err
		}

		return sendCreateDhcpOuiCmdToDHCPAgent(dhcpoui)
	}); err != nil {
		return nil, err
	}

	return dhcpoui, nil
}

func sendCreateDhcpOuiCmdToDHCPAgent(dhcpoui *resource.DhcpOui) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateOui,
		&pbdhcpagent.CreateOuiRequest{
			Oui:          dhcpoui.Oui,
			Organization: dhcpoui.Organization,
		})
}

func (d *DhcpOuiService) List(ctx *restresource.Context) (interface{}, error) {
	listCtx := genGetOUIContext(ctx)
	var ouis []*resource.DhcpOui
	var ouiCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableDhcpOui,
				listCtx.countSql); err != nil {
				return err
			} else {
				ouiCount = int(count)
			}
		}

		return tx.FillEx(&ouis, listCtx.sql, listCtx.params...)
	}); err != nil {
		return nil, err
	}
	setPagination(ctx, listCtx.hasPagination, ouiCount)
	return ouis, nil
}

type listOUIContext struct {
	countSql      string
	sql           string
	params        []interface{}
	hasFilterOUI  bool
	hasPagination bool
}

func genGetOUIContext(ctx *restresource.Context) listOUIContext {
	listCtx := listOUIContext{}
	if value, ok := util.GetFilterValueWithEqModifierFromFilters(FieldOUI,
		ctx.GetFilters()); ok {
		listCtx.hasFilterOUI = true
		listCtx.sql = "select * from gr_dhcp_oui where oui = $1"
		listCtx.params = append(listCtx.params, value)
	} else {
		listCtx.sql = "select * from gr_dhcp_oui"
	}

	listCtx.countSql = strings.Replace(listCtx.sql, "*", "count(*)", 1)
	if listCtx.hasFilterOUI == false {
		listCtx.sql += " order by oui"
		if pagination := ctx.GetPagination(); pagination.PageSize > 0 &&
			pagination.PageNum > 0 {
			listCtx.hasPagination = true
			listCtx.sql += " limit $1 offset $2"
			listCtx.params = append(listCtx.params, pagination.PageSize)
			listCtx.params = append(listCtx.params, (pagination.PageNum-1)*pagination.PageSize)
		}
	}

	return listCtx
}

func (d *DhcpOuiService) Get(dhcpouiID string) (restresource.Resource, error) {
	var dhcpouis []*resource.DhcpOui
	dhcpoui, err := restdb.GetResourceWithID(db.GetDB(), dhcpouiID, &dhcpouis)
	if err != nil {
		return nil, err
	}

	return dhcpoui.(*resource.DhcpOui), nil
}

func (d *DhcpOuiService) Update(dhcpoui *resource.DhcpOui) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableDhcpOui,
			map[string]interface{}{resource.SqlDhcpOuiOrg: dhcpoui.Organization},
			map[string]interface{}{restdb.IDField: dhcpoui.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found oui %s", dhcpoui.GetID())
		}
		return sendUpdateDhcpOuiCmdToDHCPAgent(dhcpoui)
	}); err != nil {
		return nil, err
	}

	return dhcpoui, nil
}

func sendUpdateDhcpOuiCmdToDHCPAgent(dhcpoui *resource.DhcpOui) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateOui,
		&pbdhcpagent.UpdateOuiRequest{
			Oui:          dhcpoui.Oui,
			Organization: dhcpoui.Organization,
		})
}

func (d *DhcpOuiService) Delete(dhcpouiId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableDhcpOui, map[string]interface{}{
			restdb.IDField: dhcpouiId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found oui %s", dhcpouiId)
		}

		return sendDeleteDhcpOuiCmdToDHCPAgent(dhcpouiId)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteDhcpOuiCmdToDHCPAgent(dhcpouiId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteOui,
		&pbdhcpagent.DeleteOuiRequest{
			Oui: dhcpouiId,
		})
}
