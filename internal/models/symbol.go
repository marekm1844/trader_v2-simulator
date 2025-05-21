package models

import (
	"fmt"
	"strings"
)

// Symbol represents a trading pair (e.g., SOL-USD)
type Symbol struct {
	Base  string `json:"base"`  // Base currency (e.g., SOL)
	Quote string `json:"quote"` // Quote currency (e.g., USD)
}

// NewSymbol creates a new Symbol from a string representation (e.g., "SOL-USD")
func NewSymbol(symbol string) (*Symbol, error) {
	parts := strings.Split(symbol, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid symbol format: %s, expected BASE-QUOTE", symbol)
	}

	base := strings.ToUpper(parts[0])
	quote := strings.ToUpper(parts[1])

	if base == "" || quote == "" {
		return nil, fmt.Errorf("invalid symbol format: %s, base and quote cannot be empty", symbol)
	}

	return &Symbol{
		Base:  base,
		Quote: quote,
	}, nil
}

// String returns the standard string representation (e.g., "SOL-USD")
func (s Symbol) String() string {
	return fmt.Sprintf("%s-%s", s.Base, s.Quote)
}

// PolygonFormat returns the symbol formatted for Polygon API (e.g., "X:SOLUSD")
func (s Symbol) PolygonFormat() string {
	return fmt.Sprintf("X:%s%s", s.Base, s.Quote)
}

// Filename returns the filename for this symbol (e.g., "SOL-USD.json")
func (s Symbol) Filename() string {
	return fmt.Sprintf("%s.json", s.String())
}