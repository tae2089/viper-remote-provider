# Viper Remote Provider

A custom remote configuration provider for [Viper](https://github.com/spf13/viper) that enables fetching and watching configuration files from remote sources like GitHub.

## Features

- 🔄 **Remote Configuration**: Fetch configuration files from GitHub repositories
- 👀 **Watch Mode**: Automatic detection of configuration changes via polling
- 🔐 **Authentication**: Support for both Personal Access Tokens and GitHub App installation
- 🚀 **Easy Integration**: Seamless integration with Viper's remote configuration API
- ⚡ **Efficient Polling**: Uses ETag-based change detection to minimize API calls

## Supported Providers

- ✅ **GitHub**: Full support with authentication and watch mode
- 🚧 **S3**: Coming soon (directory structure present)
- 🔌 **Fallback**: Automatically falls back to Viper's default remote providers (etcd, etcd3, consul, firestore, nats)

## Installation

```bash
go get github.com/tae2089/viper-remote-provider
```

## Prerequisites

For GitHub provider:
- A GitHub repository containing your configuration file(s)
- GitHub Personal Access Token or GitHub App credentials
- Set `GITHUB_TOKEN` environment variable (if using token authentication)

## Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "os"
    "time"

    "github.com/spf13/viper"
    viper_remote_provider "github.com/tae2089/viper-remote-provider"
    "github.com/tae2089/viper-remote-provider/provider/github"
)

func main() {
    // 1. Set up GitHub provider options
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        log.Fatal("GITHUB_TOKEN environment variable is required")
    }

    option := &github.Option{
        Owner:           "your-username",     // GitHub owner (user or org)
        Repository:      "config-repo",       // Repository name
        Branch:          "main",              // Branch name
        Path:            "config.yaml",       // Config file path in repo
        Token:           token,
        PollingInterval: 10 * time.Second,    // Check for changes every 10s
    }

    // 2. Register the provider with Viper
    err := viper_remote_provider.RegisterGithubProvider(option)
    if err != nil {
        log.Fatalf("Error registering GitHub provider: %v", err)
    }

    // 3. Add remote provider
    err = viper.AddRemoteProvider("github", "github.com", "config.yaml")
    if err != nil {
        log.Fatalf("Error adding remote provider: %v", err)
    }

    viper.SetConfigType("yaml") // or "json", "toml", etc.

    // 4. Read initial configuration
    err = viper.ReadRemoteConfig()
    if err != nil {
        log.Fatalf("Error reading remote config: %v", err)
    }

    // 5. Access your configuration
    fmt.Printf("Configuration: %v\n", viper.AllSettings())
}
```

### Watch for Configuration Changes

```go
func main() {
    // ... setup code from above ...

    // Read initial config
    err := viper.ReadRemoteConfig()
    if err != nil {
        log.Fatalf("Error reading remote config: %v", err)
    }

    // Start watching for changes
    viper.GetViper().WatchRemoteConfigOnChannel()

    // Your application continues to run
    // Configuration will be automatically updated when changes are detected
    select {}
}
```

### Using GitHub App Authentication

```go
option := &github.Option{
    Owner:           "your-org",
    Repository:      "config-repo",
    Branch:          "main",
    Path:            "config.yaml",
    PemFilePath:     "/path/to/github-app.pem",  // GitHub App private key
    PollingInterval: 30 * time.Second,
}

err := viper_remote_provider.RegisterGithubProvider(option)
if err != nil {
    log.Fatalf("Error registering GitHub provider: %v", err)
}
```

## API Reference

### GitHub Provider Options

```go
type Option struct {
    Owner           string        // Repository owner (username or organization)
    Repository      string        // Repository name
    Branch          string        // Branch to fetch from (default: "main")
    Path            string        // Path to config file in repository
    Token           string        // GitHub Personal Access Token
    PemFilePath     string        // GitHub App private key file path (alternative to Token)
    PollingInterval time.Duration // Interval for checking changes (default: 60s)
}
```

### Functions

#### `RegisterGithubProvider(option *github.Option) error`

Initializes and registers the GitHub provider with Viper.

**Parameters:**
- `option`: GitHub provider configuration

**Returns:**
- `error`: Returns an error if registration fails (e.g., invalid options)

**Example:**
```go
err := viper_remote_provider.RegisterGithubProvider(&github.Option{
    Owner:      "tae2089",
    Repository: "config",
    Branch:     "main",
    Path:       "config.yaml",
    Token:      os.Getenv("GITHUB_TOKEN"),
})
if err != nil {
    log.Fatalf("Error: %v", err)
}
```

## Configuration File Format

The provider supports all configuration formats that Viper supports:
- YAML (`.yaml`, `.yml`)
- JSON (`.json`)
- TOML (`.toml`)
- HCL (`.hcl`)
- Properties (`.properties`)
- ENV (`.env`)

Make sure to call `viper.SetConfigType()` with the appropriate format.

## How It Works

### Initial Configuration Load

1. `RegisterGithubProvider()` creates a GitHub `ConfigManager` and registers it with Viper
2. `viper.AddRemoteProvider()` sets up the remote provider endpoint
3. `viper.ReadRemoteConfig()` fetches the initial configuration from GitHub
4. The content is decoded and merged into Viper's configuration

### Watch Mode

1. `WatchRemoteConfigOnChannel()` starts a background goroutine
2. A ticker polls GitHub at the specified `PollingInterval`
3. Uses ETag headers to detect changes efficiently
4. When a change is detected (ETag mismatch), new content is fetched
5. Viper automatically updates the configuration without restarting your app

## Architecture

```
┌─────────────────┐
│  Your App       │
└────────┬────────┘
         │
         ├─ viper.ReadRemoteConfig()
         ├─ viper.WatchRemoteConfigOnChannel()
         │
┌────────▼───────────────────────┐
│  Viper                         │
│  - RemoteConfig interface      │
└────────┬───────────────────────┘
         │
┌────────▼───────────────────────┐
│  remoteConfigProvider          │
│  - Get()                       │
│  - Watch()                     │
│  - WatchChannel()              │
└────────┬───────────────────────┘
         │
┌────────▼───────────────────────┐
│  GitHub ConfigManager          │
│  - Get(path) -> []byte         │
│  - Watch() -> <-chan Response  │
└────────┬───────────────────────┘
         │
┌────────▼───────────────────────┐
│  GitHub API                    │
│  - GetContents()               │
│  - ETag-based change detection │
└────────────────────────────────┘
```

## Examples

See the [example/github](example/github) directory for complete working examples.

## Best Practices

1. **Polling Interval**: Balance between responsiveness and API rate limits
   - Development: 10-30 seconds
   - Production: 60-300 seconds

2. **Error Handling**: Always check errors when reading remote config
   ```go
   if err := viper.ReadRemoteConfig(); err != nil {
       log.Fatalf("Failed to read config: %v", err)
   }
   ```

3. **Token Security**: Never hardcode tokens; use environment variables
   ```go
   token := os.Getenv("GITHUB_TOKEN")
   ```

4. **Branch Strategy**: Use stable branches for production configurations
   ```go
   option.Branch = "production" // instead of "main"
   ```

## Troubleshooting

### "GITHUB_TOKEN environment variable is required"
- Set the `GITHUB_TOKEN` environment variable with a valid GitHub Personal Access Token
- Or use `PemFilePath` for GitHub App authentication

### Configuration not updating
- Verify `PollingInterval` is set appropriately
- Check that `WatchRemoteConfigOnChannel()` is called
- Ensure your app doesn't exit immediately (use `select {}` or similar)

### Rate limit errors
- Increase `PollingInterval` to reduce API calls
- Use GitHub App authentication for higher rate limits
- The provider uses ETag headers to minimize unnecessary fetches

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the same license as the Viper project.

## Related Projects

- [Viper](https://github.com/spf13/viper) - Go configuration with fangs
- [go-github](https://github.com/google/go-github) - Go library for accessing the GitHub API

## Roadmap

- [ ] S3 provider implementation
- [ ] Support for multiple configuration files
- [ ] Caching layer for reduced API calls
- [ ] Webhook-based updates (instead of polling)
- [ ] Support for private repositories with SSH keys
