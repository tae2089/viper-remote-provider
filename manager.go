package viper_remote_provider

import (
	"github.com/sagikazarmark/crypt/config"
)

type viperConfigManager interface {
	Get(key string) ([]byte, error)
	Watch(key string, stop chan bool) <-chan *config.Response
}
