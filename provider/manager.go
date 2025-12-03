package provider

import (
	"github.com/sagikazarmark/crypt/config"
)

type ViperConfigManager interface {
	Get(key string) ([]byte, error)
	Watch(key string, stop chan bool) <-chan *config.Response
}
