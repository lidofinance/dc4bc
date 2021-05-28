package kafka_storage

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

func GetTLSConfig(trustStorePath string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(trustStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trustStorePath: %w", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	config := &tls.Config{
		RootCAs: caCertPool,
	}
	return config, nil
}
