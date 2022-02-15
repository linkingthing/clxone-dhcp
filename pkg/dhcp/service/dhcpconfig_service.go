package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type DhcpConfigService struct {
}

func NewDhcpConfigService() *DhcpConfigService {
	return &DhcpConfigService{}
}

func (d *DhcpConfigService) CreateDefaultDhcpConfig() error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableDhcpConfig, nil); err != nil {
			return fmt.Errorf("check dhcp config failed: %s", err.Error())
		} else if exists == false {
			if _, err := tx.Insert(resource.DefaultDhcpConfig); err != nil {
				return fmt.Errorf("insert default dhcp config failed: %s", err.Error())
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (d *DhcpConfigService) List() (interface{}, error) {
	var configs []*resource.DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &configs)
	}); err != nil {
		return nil, err
	}
	return configs, nil
}

func (d *DhcpConfigService) Get(configID string) (restresource.Resource, error) {
	var configs []*resource.DhcpConfig
	config, err := restdb.GetResourceWithID(db.GetDB(), configID, &configs)
	if err != nil {
		return nil, err
	}
	return config.(*resource.DhcpConfig), nil
}

func (d *DhcpConfigService) Update(config *resource.DhcpConfig) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableDhcpConfig, map[string]interface{}{
			resource.SqlColumnValidLifetime:     config.ValidLifetime,
			resource.SqlColumnMaxValidLifetime:  config.MaxValidLifetime,
			resource.SqlColumnMinValidLifetime:  config.MinValidLifetime,
			resource.SqlDhcpConfigDomainServers: config.DomainServers,
		}, map[string]interface{}{restdb.IDField: config.GetID()})
		return err
	}); err != nil {
		return nil, err
	}
	return config, nil
}
