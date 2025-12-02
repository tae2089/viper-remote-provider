package github

type GithubProvider struct {
	provider      string
	endpoint      string
	path          string
	secretKeyring string
}

func DefaultRemoteProvider() *GithubProvider {
	return &GithubProvider{provider: "github", endpoint: "localhost", path: "", secretKeyring: ""}
}

func (rp GithubProvider) Provider() string {
	return rp.provider
}

func (rp GithubProvider) Endpoint() string {
	return rp.endpoint
}

func (rp GithubProvider) Path() string {
	return rp.path
}

func (rp GithubProvider) SecretKeyring() string {
	return rp.secretKeyring
}
