package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type DhcpConfigService struct {
}

func NewDhcpConfigService() (*DhcpConfigService, error) {
	if err := CreateDefaultDhcpConfig(); err != nil {
		return nil, err
	}

	return &DhcpConfigService{}, nil
}

func CreateDefaultDhcpConfig() error {
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

func (d *DhcpConfigService) List() ([]*resource.DhcpConfig, error) {
	var configs []*resource.DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &configs)
	}); err != nil {
		return nil, fmt.Errorf("list dhcp config failed:%s", err.Error())
	}

	return configs, nil
}

func (d *DhcpConfigService) Get(id string) (*resource.DhcpConfig, error) {
	var configs []*resource.DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &configs)
	}); err != nil {
		return nil, fmt.Errorf("get dhcp config %s failed:%s", id, err.Error())
	} else if len(configs) == 0 {
		return nil, fmt.Errorf("no found dhcp config %s", id)
	}

	return configs[0], nil
}

func (d *DhcpConfigService) Update(config *resource.DhcpConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validate config config params failed: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableDhcpConfig, map[string]interface{}{
			resource.SqlColumnValidLifetime:    config.ValidLifetime,
			resource.SqlColumnMaxValidLifetime: config.MaxValidLifetime,
			resource.SqlColumnMinValidLifetime: config.MinValidLifetime,
			resource.SqlColumnDomainServers:    config.DomainServers,
		}, map[string]interface{}{restdb.IDField: config.GetID()})
		return err
	}); err != nil {
		return fmt.Errorf("update dhcp config %s failed:%s", config.GetID(), err.Error())
	}

	return nil
}
