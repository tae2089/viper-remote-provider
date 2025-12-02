package github

import "time"

type Option struct {
	Owner           string
	Repository      string
	Branch          string
	Path            string
	Token           string
	PemFilePath     string
	PollingInterval time.Duration // Watch polling interval (default: 60 seconds)
}
