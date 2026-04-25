package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Loader struct {
	sources []Source
}

func NewLoader(sources []Source) Loader {
	return Loader{sources: sources}
}

func (loader Loader) Load() (Config, error) {
	return LoadWithSources(loader.sources...)
}

func LoadWithSources(sources ...Source) (Config, error) {
	viperInstance := viper.New()

	for _, source := range sources {
		if err := source.Apply(viperInstance); err != nil {
			return Config{}, fmt.Errorf("apply %s source: %w", source.Name(), err)
		}
	}

	var cfg Config
	if err := viperInstance.Unmarshal(&cfg, func(decoderConfig *mapstructure.DecoderConfig) {
		decoderConfig.TagName = "mapstructure"
		decoderConfig.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)
	}); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
