package vaultutil

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func DecodeSecret[T any](manager *Manager, path string) (T, error) {
	var result T

	generic, err := manager.GetClient().Logical().Read(path)
	if err != nil {
		return result, fmt.Errorf("read generic data: %w", err)
	}

	config := &mapstructure.DecoderConfig{
		Result:     &result,
		TagName:    "vault",
		ErrorUnset: true,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return result, err
	}

	err = decoder.Decode(generic.Data["data"])
	if err != nil {
		return result, err
	}

	return result, err
}
