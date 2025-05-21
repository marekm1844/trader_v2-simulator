# Data Loader Module

This package provides a data loading and parsing module for the cryptocurrency trading simulator application. It handles loading historical price data from JSON files, parsing and validating the data, and providing a clean interface for accessing the data.

## Key Features

- Loads historical price data from JSON files in the data/ directory
- Parses and validates data into model structures
- Provides efficient access to data by symbol, timeframe, and date range
- Handles both daily and hourly candle data
- Implements caching for better performance
- Provides robust error handling and reporting

## Main Components

### Interfaces

- `DataLoader`: The main interface for loading and accessing historical price data
- `Option`: Functional options for configuring the data loader

### Implementation

- `FileDataLoader`: Concrete implementation of the DataLoader interface
- `Cache`: In-memory cache for candle data
- `Factory`: Factory for creating and configuring DataLoader instances

## Usage

```go
import (
    "github.com/marekmajdak/trader_v2-simulator/internal/dataloader"
    "github.com/marekmajdak/trader_v2-simulator/internal/models"
    "time"
)

// Create a new data loader
factory := dataloader.NewFactory("./data")
loader, err := factory.CreateLoader()
if err != nil {
    // Handle error
}

// Load metadata
if err := loader.LoadMetadata(); err != nil {
    // Handle error
}

// Load candles for a specific symbol and timeframe
if err := loader.LoadCandles("SOL-USD", models.TimeFrameDaily); err != nil {
    // Handle error
}

// Get candles for a specific date range
startDate := time.Date(2025, 4, 25, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
candles, err := loader.GetCandles("SOL-USD", models.TimeFrameDaily, &startDate, &endDate)
if err != nil {
    // Handle error
}

// Work with the candles
for _, candle := range candles {
    // Process each candle
    fmt.Printf("Date: %s, Close: %.2f\n", candle.DateString(), candle.Close)
}
```

## Advanced Configuration

The data loader can be configured with various options:

```go
// Create a loader with custom options
loader, err := dataloader.NewFactory("./data",
    dataloader.WithCacheEnabled(true),
    dataloader.WithPreloadAll(),
).CreateLoader()
```

Available options:
- `WithCacheEnabled`: Enable or disable the cache
- `WithDataDirectory`: Set the data directory
- `WithPreloadAll`: Load all available data into cache at initialization
- `WithCacheTTL`: Set the TTL for cached data (placeholder for future implementation)
- `WithValidator`: Set a custom validation function (placeholder for future implementation)

## Error Handling

The package provides several error types to help with error handling:

- `ErrDataNotLoaded`: When attempting to access data that hasn't been loaded
- `ErrSymbolNotFound`: When the requested symbol is not found
- `ErrTimeframeNotSupported`: When the requested timeframe is not supported
- `ErrNoDataInRange`: When no data is available in the specified date range
- `ErrInvalidDataFormat`: When the data file has an invalid format
- `ErrDataInconsistency`: When data validation fails

## Performance Considerations

- Data is cached in memory for efficient access
- The cache is thread-safe for concurrent access
- Data validation is performed to ensure data integrity
- The module is designed to work with large datasets efficiently