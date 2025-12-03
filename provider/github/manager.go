package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v79/github"
	"github.com/sagikazarmark/crypt/config"
)

type ConfigManager struct {
	client *github.Client
	option *Option
}

func NewGithubConfigManager(option *Option) (*ConfigManager, error) {
	var client *github.Client
	if option.PemFilePath != "" {
		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 1, 99, option.PemFilePath)
		if err != nil {
			return nil, err
		}
		client = github.NewClient(&http.Client{Transport: itr})
	} else {
		client = github.NewClient(nil).WithAuthToken(option.Token)
	}

	if option.PollingInterval == 0 {
		option.PollingInterval = 60 * time.Second
	}

	return &ConfigManager{option: option, client: client}, nil
}

func (cm *ConfigManager) Get(dataId string) ([]byte, error) {
	ctx := context.Background()
	opts := &github.RepositoryContentGetOptions{}
	if cm.option.Branch != "" {
		opts.Ref = cm.option.Branch
	}
	content, _, _, err := cm.client.Repositories.GetContents(ctx, cm.option.Owner, cm.option.Repository, cm.option.Path, opts)
	if err != nil {
		return nil, err
	}

	if content == nil {
		return nil, nil
	}

	decodedContent, err := content.GetContent()
	if err != nil {
		return nil, err
	}

	return []byte(decodedContent), nil
}

func (cm *ConfigManager) Watch(dataId string, stop chan bool) <-chan *config.Response {
	respChan := make(chan *config.Response)

	go func() {
		ctx := context.Background()
		var etag string
		pollInterval := cm.option.PollingInterval

		opts := &github.RepositoryContentGetOptions{}
		if cm.option.Branch != "" {
			opts.Ref = cm.option.Branch
		}

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		// Initial fetch
		etag = cm.fetchAndNotify(ctx, opts, etag, respChan)

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Println("Polling for changes...")
				etag = cm.fetchAndNotify(ctx, opts, etag, respChan)
			}
		}
	}()

	return respChan
}

func (cm *ConfigManager) fetchAndNotify(ctx context.Context, opts *github.RepositoryContentGetOptions, etag string, respChan chan<- *config.Response) string {
	content, _, resp, err := cm.client.Repositories.GetContents(ctx, cm.option.Owner, cm.option.Repository, cm.option.Path, opts)

	if err != nil {
		respChan <- &config.Response{Value: nil, Error: err}
		// Implement exponential backoff on errors
		time.Sleep(time.Second * 5)
		return etag
	}

	// Update ETag from response headers
	if resp != nil && resp.Response != nil && resp.Response.Header != nil {
		newEtag := resp.Response.Header.Get("ETag")

		// If ETag changed (or first fetch), content has changed
		if newEtag != "" && newEtag != etag {
			if content != nil {
				decodedContent, err := content.GetContent()
				if err != nil {
					respChan <- &config.Response{Value: nil, Error: err}
				} else {
					fmt.Println("changed!!!")
					respChan <- &config.Response{Value: []byte(decodedContent), Error: nil}
					return newEtag
				}
			}
		}
	}
	return etag
}
