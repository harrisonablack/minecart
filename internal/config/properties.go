package config

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"strconv"

	"github.com/magiconair/properties"
)

func mergeProperties(path string, values map[string]any) ([]byte, error) {
	p := properties.NewProperties()
	// Java properties are literal values, not Spring-style ${...} expressions.
	p.DisableExpansion = true
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read properties: %w", err)
	}
	if err := p.Load(data, properties.UTF8); err != nil {
		return nil, fmt.Errorf("parse properties: %w", err)
	}
	for key, value := range values {
		if key == "" {
			return nil, fmt.Errorf("property key must not be empty")
		}
		text, err := propertyValue(value)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", key, err)
		}
		if _, _, err := p.Set(key, text); err != nil {
			return nil, fmt.Errorf("set property %q: %w", key, err)
		}
	}
	p.Sort()
	var out bytes.Buffer
	if _, err := p.Write(&out, properties.UTF8); err != nil {
		return nil, fmt.Errorf("encode properties: %w", err)
	}
	// Refuse a lossy serialization before replacing the file. In particular,
	// the library writer does not escape every possible special key (such as
	// a leading comment marker). Keep that codec responsibility in the library.
	loader := properties.Loader{Encoding: properties.UTF8, DisableExpansion: true}
	check, err := loader.LoadBytes(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("check encoded properties: %w", err)
	}
	if !maps.Equal(p.Map(), check.Map()) {
		return nil, errors.New("properties library cannot serialize these keys and values without loss")
	}
	return out.Bytes(), nil
}

func propertyValue(value any) (string, error) {
	switch v := value.(type) {
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		// JSON numbers arrive as floats, including integer settings. Avoid
		// exponent notation that Java's integer parsers do not accept.
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(value), nil
	default:
		return "", fmt.Errorf("expected a string, number, or boolean, got %T", value)
	}
}
