package github

import (
	"fmt"
	"time"
)

type Option struct {
	Owner           string
	Repository      string
	Branch          string
	Path            string
	Token           string
	PemFilePath     string
	PollingInterval time.Duration // Watch polling interval (default: 60 seconds)
}

// Validate는 ProviderOptions 인터페이스 구현
func (o *Option) Validate() error {
	if o.Owner == "" {
		return fmt.Errorf("owner is required")
	}
	if o.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if o.Path == "" {
		return fmt.Errorf("path is required")
	}
	if o.Token == "" && o.PemFilePath == "" {
		return fmt.Errorf("either token or pem file path is required")
	}
	return nil
}
