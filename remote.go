package viper_remote_provider

import (
	"bytes"
	"io"
	"os"
	"strings"

	crypt "github.com/sagikazarmark/crypt/config"
	"github.com/spf13/viper"
	"github.com/tae2089/viper-remote-provider/provider/github"
)

type remoteConfigProvider struct {
	GithubConfigManager *github.ConfigManager
}

func SetOptions(option *github.Option) {
	m, _ := github.NewGithubConfigManager(option)
	viper.SupportedRemoteProviders = append(viper.SupportedRemoteProviders, "github")
	viper.RemoteConfig = &remoteConfigProvider{GithubConfigManager: m}
}

func (rc remoteConfigProvider) Get(rp viper.RemoteProvider) (io.Reader, error) {
	var cm viperConfigManager
	var err error
	switch rp.Provider() {
	case "github":
		cm = rc.GithubConfigManager
	default:
		cm, err = getConfigManager(rp)
	}

	if err != nil {
		return nil, err
	}
	b, err := cm.Get(rp.Path())
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (rc remoteConfigProvider) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	var cm viperConfigManager
	var err error
	switch rp.Provider() {
	case "github":
		cm = rc.GithubConfigManager
	default:
		cm, err = getConfigManager(rp)
	}
	if err != nil {
		return nil, err
	}
	b, err := cm.Get(rp.Path())
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (rc remoteConfigProvider) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	var cm viperConfigManager
	switch rp.Provider() {
	case "github":
		cm = rc.GithubConfigManager
	default:
		cm, _ = getConfigManager(rp)
	}
	quit := make(chan bool)
	quitwc := make(chan bool)
	viperResponsCh := make(chan *viper.RemoteResponse)
	cryptoResponseCh := cm.Watch(rp.Path(), quit)
	// need this function to convert the Channel response form crypt.Response to viper.Response
	go func(cr <-chan *crypt.Response, vr chan<- *viper.RemoteResponse, quitwc <-chan bool, quit chan<- bool) {
		for {
			select {
			case <-quitwc:
				quit <- true
				return
			case resp := <-cr:
				vr <- &viper.RemoteResponse{
					Error: resp.Error,
					Value: resp.Value,
				}
			}
		}
	}(cryptoResponseCh, viperResponsCh, quitwc, quit)

	return viperResponsCh, quitwc
}

func getConfigManager(rp viper.RemoteProvider) (crypt.ConfigManager, error) {
	var cm crypt.ConfigManager
	var err error

	endpoints := strings.Split(rp.Endpoint(), ";")
	if rp.SecretKeyring() != "" {
		var kr *os.File
		kr, err = os.Open(rp.SecretKeyring())
		if err != nil {
			return nil, err
		}
		defer kr.Close()
		switch rp.Provider() {
		case "etcd":
			cm, err = crypt.NewEtcdConfigManager(endpoints, kr)
		case "etcd3":
			cm, err = crypt.NewEtcdV3ConfigManager(endpoints, kr)
		case "firestore":
			cm, err = crypt.NewFirestoreConfigManager(endpoints, kr)
		case "nats":
			cm, err = crypt.NewNatsConfigManager(endpoints, kr)
		default:
			cm, err = crypt.NewConsulConfigManager(endpoints, kr)
		}
	} else {
		switch rp.Provider() {
		case "etcd":
			cm, err = crypt.NewStandardEtcdConfigManager(endpoints)
		case "etcd3":
			cm, err = crypt.NewStandardEtcdV3ConfigManager(endpoints)
		case "firestore":
			cm, err = crypt.NewStandardFirestoreConfigManager(endpoints)
		case "nats":
			cm, err = crypt.NewStandardNatsConfigManager(endpoints)
		default:
			cm, err = crypt.NewStandardConsulConfigManager(endpoints)
		}
	}
	if err != nil {
		return nil, err
	}
	return cm, nil
}
