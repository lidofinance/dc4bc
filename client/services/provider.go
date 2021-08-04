package services

import (
	"github.com/lidofinance/dc4bc/client/config"
	"github.com/lidofinance/dc4bc/client/modules/keystore"
	"github.com/lidofinance/dc4bc/client/services/client"
	"github.com/lidofinance/dc4bc/storage"
)

var provider ServiceProvider

type ServiceProvider struct {
	baseClientService *client.BaseClientService
}

// Init services
func (p *ServiceProvider) Init(config *config.Config, storage storage.Storage, ks keystore.KeyStore) error {
	var err error

	p.baseClientService, err = client.Init(config, storage, ks)

	if err != nil {
		return err
	}

	return nil
}

func (p *ServiceProvider) BaseClientService() *client.BaseClientService {
	return p.baseClientService
}

func App() *ServiceProvider {
	return &provider
}
