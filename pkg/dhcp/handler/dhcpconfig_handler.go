package handler

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	DefaultIdentify = "dhcpglobalconfig"
)

type DhcpConfigHandler struct {
}

func NewDhcpConfigHandler() *DhcpConfigHandler {
	return &DhcpConfigHandler{}
}

func (d *DhcpConfigHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var configs []*resource.DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(nil, &configs); err != nil {
			return err
		}

		if len(configs) == 0 {
			config := &resource.DhcpConfig{
				Identify:         DefaultIdentify,
				MinValidLifetime: resource.DefaultMinValidLifetime,
				MaxValidLifetime: resource.DefaultMaxValidLifetime,
				ValidLifetime:    resource.DefaultValidLifetime,
			}
			tx.Insert(config)
			configs = append(configs, config)
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list global config from db failed: %s", err.Error()))
	}

	return configs, nil
}

func (d *DhcpConfigHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	configID := ctx.Resource.(*resource.DhcpConfig).GetID()
	var configs []*resource.DhcpConfig
	config, err := restdb.GetResourceWithID(db.GetDB(), configID, &configs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get global config %s from db failed: %s", configID, err.Error()))
	}

	return config.(*resource.DhcpConfig), nil
}

func (d *DhcpConfigHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	config := ctx.Resource.(*resource.DhcpConfig)
	if err := config.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update global config params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableDhcpConfig, map[string]interface{}{
			"valid_lifetime":     config.ValidLifetime,
			"max_valid_lifetime": config.MaxValidLifetime,
			"min_valid_lifetime": config.MinValidLifetime,
			"domain_servers":     config.DomainServers,
		}, map[string]interface{}{restdb.IDField: config.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update global config %s failed: %s", config.GetID(), err.Error()))
	}

	return config, nil
}
