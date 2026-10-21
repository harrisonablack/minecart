package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	yamlv3 "go.yaml.in/yaml/v3"
)

func mergeYAML(path string, values map[string]any) ([]byte, error) {
	// The lookup delimiter is only for Koanf's index. All operations here use
	// nested maps, preserving literal dotted keys alongside nested equivalents.
	// This instance contains only this destination's values, not app config.
	k := koanf.New(".")
	parser := singleDocumentYAML{yaml.Parser()}
	if err := k.Load(file.Provider(path), parser); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load YAML: %w", err)
	}
	// An empty provider delimiter keeps keys literal instead of unflattening
	// them as paths. Koanf supplies the native recursive merge semantics.
	if err := k.Load(confmap.Provider(values, ""), nil); err != nil {
		return nil, fmt.Errorf("merge YAML: %w", err)
	}
	// Marshal uses the nested raw map, not the flattened key listing.
	data, err := k.Marshal(parser)
	if err != nil {
		return nil, fmt.Errorf("encode YAML: %w", err)
	}
	return data, nil
}

// Koanf's YAML parser reads the first document only. Check for trailing
// documents so serializing a managed file cannot silently discard them.
// Parsing and merging the configuration itself still belong to Koanf.
type singleDocumentYAML struct {
	koanf.Parser
}

func (p singleDocumentYAML) Unmarshal(data []byte) (map[string]any, error) {
	values, err := p.Parser.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	decoder := yamlv3.NewDecoder(bytes.NewReader(data))
	var document any
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return values, nil
		}
		return nil, err
	}
	if err := decoder.Decode(&document); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("multiple YAML documents are not supported")
	}
	return values, nil
}
