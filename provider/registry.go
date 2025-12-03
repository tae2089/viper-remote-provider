package provider

import (
	"fmt"
	"sync"
)

var (
	providerRegistry = &ProviderRegistry{
		providers: make(map[Type]*ProviderRegistration),
	}
)

type ProviderRegistration struct {
	Factory Factory
	Manager ViperConfigManager
}

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[Type]*ProviderRegistration
}

func Register(providerType Type, options Options, factory Factory) error {
	return providerRegistry.RegisterProvider(providerType, options, factory)
}

func IsRegistered(providerType Type) bool {
	return providerRegistry.IsRegistered(providerType)
}

func GetManager(providerType Type) (ViperConfigManager, error) {
	return providerRegistry.GetManager(providerType)
}

// RegisterProvider는 새 provider를 등록
func (r *ProviderRegistry) RegisterProvider(
	providerType Type,
	options Options,
	factory Factory,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := options.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	manager, err := factory(options)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	r.providers[providerType] = &ProviderRegistration{
		Factory: factory,
		Manager: manager,
	}

	return nil
}

// GetManager는 등록된 provider의 manager를 반환
func (r *ProviderRegistry) GetManager(providerType Type) (ViperConfigManager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	registration, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerType)
	}

	return registration.Manager, nil
}

// IsRegistered는 provider가 등록되었는지 확인
func (r *ProviderRegistry) IsRegistered(providerType Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[providerType]
	return exists
}
