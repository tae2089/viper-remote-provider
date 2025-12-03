package viper_remote_provider

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	crypt "github.com/sagikazarmark/crypt/config"
	"github.com/spf13/viper"
	"github.com/tae2089/viper-remote-provider/provider"
	"github.com/tae2089/viper-remote-provider/provider/github"
)

type remoteConfigProvider struct{}

// RegisterProvider는 새 provider를 등록하고 viper에 추가
func RegisterProvider(
	providerType provider.Type,
	options provider.Options,
	factory provider.Factory,
) error {
	if err := provider.Register(providerType, options, factory); err != nil {
		return err
	}

	// viper에 provider 추가
	providerName := string(providerType)
	if !contains(viper.SupportedRemoteProviders, providerName) {
		viper.SupportedRemoteProviders = append(viper.SupportedRemoteProviders, providerName)
	}

	// 첫 provider 등록 시 RemoteConfig 설정
	if viper.RemoteConfig == nil {
		viper.RemoteConfig = &remoteConfigProvider{}
	}

	return nil
}

func (rc remoteConfigProvider) Get(rp viper.RemoteProvider) (io.Reader, error) {
	cm, err := rc.getConfigManager(rp)
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
	cm, err := rc.getConfigManager(rp)
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
	cm, err := rc.getConfigManager(rp)
	if err != nil {
		// 에러 처리를 위한 채널 생성
		errCh := make(chan *viper.RemoteResponse, 1)
		errCh <- &viper.RemoteResponse{Error: err}
		close(errCh)
		return errCh, make(chan bool)
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

func (rc remoteConfigProvider) getConfigManager(rp viper.RemoteProvider) (provider.ViperConfigManager, error) {
	providerType := provider.Type(rp.Provider())

	// Registry에서 먼저 조회
	if provider.IsRegistered(providerType) {
		return provider.GetManager(providerType)
	}

	// Registry에 없으면 기존 crypt provider 사용 (etcd, consul 등)
	return getConfigManager(rp)
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

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

// RegisterGithubProvider는 GitHub provider를 등록하는 편의 함수
func RegisterGithubProvider(options *github.Option) error {
	factory := func(opts provider.Options) (provider.ViperConfigManager, error) {
		githubOpts, ok := opts.(*github.Option)
		if !ok {
			return nil, fmt.Errorf("invalid options type for github provider")
		}
		return github.NewGithubConfigManager(githubOpts)
	}

	return RegisterProvider(provider.GitHub, options, factory)
}
