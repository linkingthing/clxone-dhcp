package db

import (
	"fmt"

	"github.com/linkingthing/clxone-utils/pbe"
	restdb "github.com/linkingthing/gorest/db"
	"github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
)

const (
	ConnStr = "user=%s password=%s host=%s port=%d database=%s sslmode=disable pool_max_conns=10"
)

var globalResources []resource.Resource

func RegisterResources(resources ...resource.Resource) {
	globalResources = append(globalResources, resources...)
}

var globalDB restdb.ResourceStore

func GetDB() restdb.ResourceStore {
	return globalDB
}

func Init(conf *config.DHCPConfig) error {
	meta, err := restdb.NewResourceMeta(globalResources)
	if err != nil {
		return err
	}

	iterator := conf.Server.DecryptIterator
	if iterator == 0 {
		iterator = 10000
	}

	decryptPassword, err := pbe.Decrypt(&pbe.DecryptContext{
		KeyFactoryBase64: conf.Server.KeyFactoryBase64,
		EncryptWorkKey:   conf.Server.EncryptWorkKey,
		EncryptPassword:  conf.DB.Password,
		Iterator:         iterator,
	})
	if err != nil {
		return err
	}

	globalDB, err = restdb.NewRStore(fmt.Sprintf(ConnStr,
		conf.DB.User, decryptPassword, conf.DB.Host, conf.DB.Port, conf.DB.Name),
		meta)
	return err
}

func GetResources(conditions map[string]interface{}, resources interface{}) error {
	return restdb.WithTx(globalDB, func(tx restdb.Transaction) error {
		return tx.Fill(conditions, resources)
	})
}
