package dataloader

import (
	"sync"
	"time"

	"github.com/marekm1844/trader_v2-simulator/internal/models"
)

// CacheKey is used to identify candles in the cache
type CacheKey struct {
	Symbol    string
	Timeframe models.TimeFrame
}

// CacheEntry stores cached candle data with metadata
type CacheEntry struct {
	Candles    []models.Candle
	LastUpdate time.Time
	IsComplete bool
}

// Cache provides an in-memory cache for candle data
type Cache struct {
	entries map[CacheKey]CacheEntry
	mu      sync.RWMutex
}

// NewCache creates a new empty cache
func NewCache() *Cache {
	return &Cache{
		entries: make(map[CacheKey]CacheEntry),
	}
}

// Get retrieves candles from the cache
func (c *Cache) Get(symbol string, timeframe models.TimeFrame) ([]models.Candle, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := CacheKey{Symbol: symbol, Timeframe: timeframe}
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	return entry.Candles, true
}

// Set stores candles in the cache
func (c *Cache) Set(symbol string, timeframe models.TimeFrame, candles []models.Candle, isComplete bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := CacheKey{Symbol: symbol, Timeframe: timeframe}
	c.entries[key] = CacheEntry{
		Candles:    candles,
		LastUpdate: time.Now(),
		IsComplete: isComplete,
	}
}

// Has checks if the cache has an entry for the given key
func (c *Cache) Has(symbol string, timeframe models.TimeFrame) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := CacheKey{Symbol: symbol, Timeframe: timeframe}
	_, ok := c.entries[key]
	return ok
}

// Remove removes an entry from the cache
func (c *Cache) Remove(symbol string, timeframe models.TimeFrame) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := CacheKey{Symbol: symbol, Timeframe: timeframe}
	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[CacheKey]CacheEntry)
}

// GetCachedSymbols returns all symbols that have cached data
func (c *Cache) GetCachedSymbols() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	symbolMap := make(map[string]struct{})
	for key := range c.entries {
		symbolMap[key.Symbol] = struct{}{}
	}

	symbols := make([]string, 0, len(symbolMap))
	for symbol := range symbolMap {
		symbols = append(symbols, symbol)
	}

	return symbols
}

// GetCachedTimeframes returns all timeframes that have cached data for a symbol
func (c *Cache) GetCachedTimeframes(symbol string) []models.TimeFrame {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var timeframes []models.TimeFrame
	for key := range c.entries {
		if key.Symbol == symbol {
			timeframes = append(timeframes, key.Timeframe)
		}
	}

	return timeframes
}