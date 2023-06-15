package service

import (
	"strings"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
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

func (d *DhcpOuiService) Create(dhcpOui *resource.DhcpOui) error {
	if err := dhcpOui.Validate(); err != nil {
		return err
	}

	dhcpOui.SetID(dhcpOui.Oui)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(dhcpOui); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
		}

		return sendCreateDhcpOuiCmdToDHCPAgent(dhcpOui)
	})
}

func sendCreateDhcpOuiCmdToDHCPAgent(dhcpOui *resource.DhcpOui) error {
	return kafka.SendDHCPCmd(kafka.CreateOui, &pbdhcpagent.CreateOuiRequest{
		Oui:          dhcpOui.Oui,
		Organization: dhcpOui.Organization,
	}, func(nodesForSucceed []string) {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
			kafka.DeleteOui, &pbdhcpagent.DeleteOuiRequest{Oui: dhcpOui.Oui}); err != nil {
			log.Errorf("create oui %s failed, rollback with nodes %v failed: %s",
				dhcpOui.Oui, nodesForSucceed, err.Error())
		}
	})
}

func (d *DhcpOuiService) List(ctx *restresource.Context) ([]*resource.DhcpOui, error) {
	listCtx := genGetOUIContext(ctx)
	var ouis []*resource.DhcpOui
	var ouiCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableDhcpOui,
				listCtx.countSql); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameOui), pg.Error(err).Error())
			} else {
				ouiCount = int(count)
			}
		}

		if err := tx.FillEx(&ouis, listCtx.sql, listCtx.params...); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameOui), pg.Error(err).Error())
		}
		return nil
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
	if !listCtx.hasFilterOUI {
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

func (d *DhcpOuiService) Get(id string) (*resource.DhcpOui, error) {
	var dhcpOuis []*resource.DhcpOui
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &dhcpOuis)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(dhcpOuis) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameOui, id)
	}

	return dhcpOuis[0], nil
}

func (d *DhcpOuiService) Update(dhcpOui *resource.DhcpOui) error {
	if err := dhcpOui.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := d.checkOuiIsReadOnly(tx, dhcpOui.GetID()); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableDhcpOui, map[string]interface{}{
			resource.SqlColumnOuiOrganization: dhcpOui.Organization,
		}, map[string]interface{}{restdb.IDField: dhcpOui.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, dhcpOui.GetID(), pg.Error(err).Error())
		}

		return sendUpdateDhcpOuiCmdToDHCPAgent(dhcpOui)
	})
}

func (d *DhcpOuiService) checkOuiIsReadOnly(tx restdb.Transaction, id string) error {
	var dhcpOuis []*resource.DhcpOui
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id},
		&dhcpOuis); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(dhcpOuis) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameOui, id)
	} else if dhcpOuis[0].IsReadOnly {
		return errorno.ErrReadOnly(id)
	} else {
		return nil
	}
}

func sendUpdateDhcpOuiCmdToDHCPAgent(dhcpoui *resource.DhcpOui) error {
	return kafka.SendDHCPCmd(kafka.UpdateOui, &pbdhcpagent.UpdateOuiRequest{
		Oui:          dhcpoui.Oui,
		Organization: dhcpoui.Organization,
	}, nil)
}

func (d *DhcpOuiService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := d.checkOuiIsReadOnly(tx, id); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableDhcpOui, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteDhcpOuiCmdToDHCPAgent(id)
	})
}

func sendDeleteDhcpOuiCmdToDHCPAgent(dhcpOuiId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteOui,
		&pbdhcpagent.DeleteOuiRequest{Oui: dhcpOuiId}, nil)
}
