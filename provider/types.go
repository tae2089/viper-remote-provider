package provider

// ProviderType은 지원되는 provider 타입
type Type string

const (
	GitHub Type = "github"
	S3     Type = "s3"
	GCS    Type = "gcs"
)

// ProviderOptions는 각 provider의 옵션 인터페이스
type Options interface {
	Validate() error
}

// ProviderFactory는 provider의 ConfigManager를 생성하는 factory 함수
type Factory func(options Options) (ViperConfigManager, error)
