package dataloader

import (
	"fmt"
	"path/filepath"
)

// Factory is responsible for creating and configuring DataLoader instances
type Factory struct {
	baseDataDir string
	options     []Option
}

// NewFactory creates a new DataLoader factory with the given base data directory
func NewFactory(baseDataDir string, options ...Option) *Factory {
	return &Factory{
		baseDataDir: baseDataDir,
		options:     options,
	}
}

// CreateLoader creates a new DataLoader instance
func (f *Factory) CreateLoader() (DataLoader, error) {
	dataDir := f.baseDataDir
	if !filepath.IsAbs(dataDir) {
		absPath, err := filepath.Abs(dataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for data directory: %w", err)
		}
		dataDir = absPath
	}

	// Create the loader
	loader := NewFileDataLoader(dataDir)

	// Apply options
	for _, option := range f.options {
		option(loader)
	}

	return loader, nil
}

// CreateLoaderWithOptions creates a new DataLoader with additional options
func (f *Factory) CreateLoaderWithOptions(additionalOptions ...Option) (DataLoader, error) {
	// Combine factory options with additional options
	allOptions := append(f.options, additionalOptions...)
	
	// Create a new factory with all options
	factory := NewFactory(f.baseDataDir, allOptions...)
	
	// Create the loader
	return factory.CreateLoader()
}