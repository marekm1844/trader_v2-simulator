package dataloader

import (
	"time"
	
	"github.com/marekm1844/trader_v2-simulator/internal/models"
)

// Option is a function that configures a FileDataLoader
type Option func(*FileDataLoader)

// WithCacheEnabled enables or disables the cache
func WithCacheEnabled(enabled bool) Option {
	return func(l *FileDataLoader) {
		if !enabled {
			// Clear the cache
			l.mu.Lock()
			defer l.mu.Unlock()
			l.candleCache = make(map[string]map[models.TimeFrame][]models.Candle)
		}
	}
}

// WithCacheTTL sets the TTL for cached data
// This is a placeholder for future implementation of cache expiration
func WithCacheTTL(ttl time.Duration) Option {
	return func(l *FileDataLoader) {
		// This would set up a cache cleanup routine
		// Not implemented in the current version
	}
}

// WithDataDirectory sets the data directory
func WithDataDirectory(dataDir string) Option {
	return func(l *FileDataLoader) {
		l.dataDir = dataDir
	}
}

// WithPreloadAll loads all available data into cache at initialization
func WithPreloadAll() Option {
	return func(l *FileDataLoader) {
		// Load metadata first
		if err := l.LoadMetadata(); err != nil {
			return
		}

		// For each symbol and timeframe, load candles
		for symbol := range l.metadataMap {
			for _, tf := range []models.TimeFrame{models.TimeFrameDaily, models.TimeFrameHourly} {
				_ = l.LoadCandles(symbol, tf) // Ignore errors
			}
		}
	}
}

// WithValidator sets a custom validation function
// This is a placeholder for future implementation of custom validation
func WithValidator(validateFunc func(loader *FileDataLoader) error) Option {
	return func(l *FileDataLoader) {
		// This would set a custom validation function
		// Not implemented in the current version
	}
}