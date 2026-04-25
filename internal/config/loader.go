package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Loader struct {
	sources []Source
}

func NewLoader(sources []Source) Loader {
	return Loader{sources: sources}
}

func (loader Loader) Load() (Config, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigType("toml")
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, source := range loader.sources {
		if err := source.Load(viperInstance); err != nil {
			return Config{}, fmt.Errorf("load config source %s: %w", source.Name(), err)
		}
	}

	var cfg Config
	if err := viperInstance.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
