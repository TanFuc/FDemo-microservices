package providers

import (
	"fmt"

	"microservices/logistic/internal/adapters/providers/ghn"
	"microservices/logistic/internal/adapters/providers/ghtk"
	"microservices/logistic/internal/adapters/providers/mock"
	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/ports"
)

// ProviderConfig holds configuration for all providers
type ProviderConfig struct {
	GHN  ghn.Config
	GHTK ghtk.Config
}

// Factory creates and manages provider instances
type Factory struct {
	providers map[domain.ProviderName]ports.Provider
}

// NewFactory creates a new provider factory
func NewFactory(cfg ProviderConfig) (*Factory, error) {
	f := &Factory{
		providers: make(map[domain.ProviderName]ports.Provider),
	}

	// Always register mock provider
	f.providers[domain.ProviderMock] = mock.NewMockProvider()

	// Register GHN provider if configured
	if cfg.GHN.Token != "" {
		ghnProvider, err := ghn.NewProvider(cfg.GHN)
		if err != nil {
			return nil, fmt.Errorf("failed to create GHN provider: %w", err)
		}
		f.providers[domain.ProviderGHN] = ghnProvider
	}

	// Register GHTK provider if configured
	if cfg.GHTK.Token != "" {
		ghtkProvider, err := ghtk.NewProvider(cfg.GHTK)
		if err != nil {
			return nil, fmt.Errorf("failed to create GHTK provider: %w", err)
		}
		f.providers[domain.ProviderGHTK] = ghtkProvider
	}

	return f, nil
}

// GetProvider returns a provider by name
func (f *Factory) GetProvider(name domain.ProviderName) (ports.Provider, error) {
	provider, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found or not configured", name)
	}
	return provider, nil
}

// ListProviders returns all registered provider names
func (f *Factory) ListProviders() []domain.ProviderName {
	names := make([]domain.ProviderName, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return names
}
