# Provider Extensibility Implementation Guide

이 문서는 향후 S3, GCS와 같은 새로운 provider를 추가하는 방법을 설명합니다.

## 개요

Provider registry 패턴을 사용하여 각 provider를 독립적으로 추가할 수 있도록 리팩토링했습니다. 새로운 provider를 추가하려면 다음 3단계만 수행하면 됩니다:

1. Provider-specific Option 구조체 생성
2. ConfigManager 구현
3. 편의 함수 추가 (선택사항)

## S3 Provider 추가 예시

### 1단계: Option 구조체 생성

`provider/s3/option.go` 파일을 생성합니다:

```go
package s3

import (
    "fmt"
    "time"
)

type Option struct {
    Bucket          string
    Region          string
    Key             string
    AccessKeyID     string
    SecretAccessKey string
    PollingInterval time.Duration
}

// Validate는 provider.Options 인터페이스 구현
func (o *Option) Validate() error {
    if o.Bucket == "" {
        return fmt.Errorf("bucket is required")
    }
    if o.Region == "" {
        return fmt.Errorf("region is required")
    }
    if o.Key == "" {
        return fmt.Errorf("key is required")
    }
    return nil
}
```

### 2단계: ConfigManager 구현

`provider/s3/manager.go` 파일을 생성합니다:

```go
package s3

import (
    "context"
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    crypt "github.com/sagikazarmark/crypt/config"
)

type ConfigManager struct {
    client *s3.Client
    option *Option
}

func NewS3ConfigManager(option *Option) (*ConfigManager, error) {
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion(option.Region),
    )
    if err != nil {
        return nil, err
    }

    client := s3.NewFromConfig(cfg)

    if option.PollingInterval == 0 {
        option.PollingInterval = 60 * time.Second
    }

    return &ConfigManager{
        client: client,
        option: option,
    }, nil
}

func (cm *ConfigManager) Get(key string) ([]byte, error) {
    result, err := cm.client.GetObject(context.TODO(), &s3.GetObjectInput{
        Bucket: aws.String(cm.option.Bucket),
        Key:    aws.String(cm.option.Key),
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()

    // Read body
    data := make([]byte, *result.ContentLength)
    _, err = result.Body.Read(data)
    if err != nil && err.Error() != "EOF" {
        return nil, err
    }

    return data, nil
}

func (cm *ConfigManager) Watch(key string, stop chan bool) <-chan *crypt.Response {
    respChan := make(chan *crypt.Response)

    go func() {
        ctx := context.Background()
        var lastModified *time.Time
        ticker := time.NewTicker(cm.option.PollingInterval)
        defer ticker.Stop()

        // Initial fetch
        lastModified = cm.fetchAndNotify(ctx, lastModified, respChan)

        for {
            select {
            case <-stop:
                return
            case <-ticker.C:
                lastModified = cm.fetchAndNotify(ctx, lastModified, respChan)
            }
        }
    }()

    return respChan
}

func (cm *ConfigManager) fetchAndNotify(
    ctx context.Context,
    lastModified *time.Time,
    respChan chan<- *crypt.Response,
) *time.Time {
    result, err := cm.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(cm.option.Bucket),
        Key:    aws.String(cm.option.Key),
    })

    if err != nil {
        respChan <- &crypt.Response{Value: nil, Error: err}
        return lastModified
    }
    defer result.Body.Close()

    // Check if modified
    if lastModified == nil || result.LastModified.After(*lastModified) {
        data := make([]byte, *result.ContentLength)
        _, err = result.Body.Read(data)
        if err != nil && err.Error() != "EOF" {
            respChan <- &crypt.Response{Value: nil, Error: err}
            return lastModified
        }

        respChan <- &crypt.Response{Value: data, Error: nil}
        return result.LastModified
    }

    return lastModified
}
```

### 3단계: 편의 함수 추가

`remote.go`에 S3 provider 등록 함수를 추가합니다:

```go
// remote.go에 추가
import (
    "github.com/tae2089/viper-remote-provider/provider"
    "github.com/tae2089/viper-remote-provider/provider/s3"
)

// RegisterS3Provider는 S3 provider를 등록하는 편의 함수
func RegisterS3Provider(options *s3.Option) error {
    factory := func(opts provider.Options) (provider.ViperConfigManager, error) {
        s3Opts, ok := opts.(*s3.Option)
        if !ok {
            return nil, fmt.Errorf("invalid options type for s3 provider")
        }
        return s3.NewS3ConfigManager(s3Opts)
    }

    return RegisterProvider(provider.S3, options, factory)
}
```

### 사용 예시

```go
package main

import (
    "log"
    
    "github.com/spf13/viper"
    vrp "github.com/tae2089/viper-remote-provider"
    "github.com/tae2089/viper-remote-provider/provider/s3"
)

func main() {
    // S3 Provider 등록
    s3Opt := &s3.Option{
        Bucket:          "my-config-bucket",
        Region:          "us-east-1",
        Key:             "config.yaml",
        AccessKeyID:     "YOUR_ACCESS_KEY",
        SecretAccessKey: "YOUR_SECRET_KEY",
    }
    
    err := vrp.RegisterS3Provider(s3Opt)
    if err != nil {
        log.Fatalf("Error registering S3 provider: %v", err)
    }
    
    // Viper에 추가
    err = viper.AddRemoteProvider("s3", "s3.amazonaws.com", "config.yaml")
    if err != nil {
        log.Fatalf("Error adding remote provider: %v", err)
    }
    
    viper.SetConfigType("yaml")
    // ... 이후 사용
}
```

## GCS Provider 추가

GCS도 유사한 패턴으로 구현할 수 있습니다:

### Option 구조체

```go
package gcs

import (
    "fmt"
    "time"
)

type Option struct {
    Bucket          string
    ProjectID       string
    Key             string
    CredentialsFile string
    PollingInterval time.Duration
}

func (o *Option) Validate() error {
    if o.Bucket == "" {
        return fmt.Errorf("bucket is required")
    }
    if o.ProjectID == "" {
        return fmt.Errorf("project ID is required")
    }
    if o.Key == "" {
        return fmt.Errorf("key is required")
    }
    return nil
}
```

### ConfigManager

```go
package gcs

import (
    "context"
    "io"
    "time"

    "cloud.google.com/go/storage"
    crypt "github.com/sagikazarmark/crypt/config"
    "google.golang.org/api/option"
)

type ConfigManager struct {
    client *storage.Client
    option *Option
}

func NewGCSConfigManager(opt *Option) (*ConfigManager, error) {
    ctx := context.Background()
    
    var client *storage.Client
    var err error
    
    if opt.CredentialsFile != "" {
        client, err = storage.NewClient(ctx, option.WithCredentialsFile(opt.CredentialsFile))
    } else {
        client, err = storage.NewClient(ctx)
    }
    
    if err != nil {
        return nil, err
    }

    if opt.PollingInterval == 0 {
        opt.PollingInterval = 60 * time.Second
    }

    return &ConfigManager{
        client: client,
        option: opt,
    }, nil
}

func (cm *ConfigManager) Get(key string) ([]byte, error) {
    ctx := context.Background()
    rc, err := cm.client.Bucket(cm.option.Bucket).Object(cm.option.Key).NewReader(ctx)
    if err != nil {
        return nil, err
    }
    defer rc.Close()

    data, err := io.ReadAll(rc)
    if err != nil {
        return nil, err
    }

    return data, nil
}

func (cm *ConfigManager) Watch(key string, stop chan bool) <-chan *crypt.Response {
    // S3와 유사한 패턴으로 구현
    // ...
}
```

### 편의 함수

`remote.go`에 추가:

```go
import (
    "github.com/tae2089/viper-remote-provider/provider"
    "github.com/tae2089/viper-remote-provider/provider/gcs"
)

func RegisterGCSProvider(options *gcs.Option) error {
    factory := func(opts provider.Options) (provider.ViperConfigManager, error) {
        gcsOpts, ok := opts.(*gcs.Option)
        if !ok {
            return nil, fmt.Errorf("invalid options type for gcs provider")
        }
        return gcs.NewGCSConfigManager(gcsOpts)
    }

    return RegisterProvider(provider.GCS, options, factory)
}
```

## 핵심 개념

### provider.Options 인터페이스
모든 provider option은 `Validate()` 메서드를 구현해야 합니다.

```go
// provider/types.go
type Options interface {
    Validate() error
}
```

### provider.ViperConfigManager 인터페이스
모든 ConfigManager는 다음 메서드를 구현해야 합니다:

```go
// provider/manager.go
type ViperConfigManager interface {
    Get(key string) ([]byte, error)
    Watch(key string, stop chan bool) <-chan *config.Response
}
```

### provider.Factory
각 provider는 factory 함수를 통해 ConfigManager를 생성합니다:

```go
// provider/types.go
type Factory func(options Options) (ViperConfigManager, error)
```

## 장점

1. **완전한 독립성**: 각 provider는 서로 독립적으로 개발 가능
2. **타입 안전성**: 각 provider가 자신만의 Option 타입 사용
3. **확장성**: 기존 코드 수정 없이 새 provider 추가 가능
4. **명확한 인터페이스**: 구현해야 할 메서드가 명확함
5. **패키지 구조**: Provider 관련 코드가 `provider` 패키지에 그룹화

## 마이그레이션 가이드

### 새로운 방식
```go
viper_remote_provider.RegisterGithubProvider(option)
```

이 방식만 지원됩니다. 에러 처리를 꼭 추가하세요:
```go
err := viper_remote_provider.RegisterGithubProvider(option)
if err != nil {
    log.Fatalf("Error: %v", err)
}
```
