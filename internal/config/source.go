package config

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Source interface {
	Name() string
	Apply(viperInstance *viper.Viper) error
}

type bytesSource struct {
	name string
	data []byte
}

type fileSource struct {
	path     string
	optional bool
}

type EnvBinding struct {
	Key     string
	EnvName string
}

type envSource struct {
	prefix   string
	bindings []EnvBinding
}

type sourceError struct {
	name string
	err  error
}

func NewEmbeddedFileSource(name string, fileSet embed.FS, fileName string) Source {
	fileContent, err := fileSet.ReadFile(fileName)
	if err != nil {
		return sourceError{
			name: name,
			err:  fmt.Errorf("read embedded file: %w", err),
		}
	}

	return bytesSource{
		name: name,
		data: fileContent,
	}
}

func (source bytesSource) Name() string {
	return source.name
}

func (source bytesSource) Apply(viperInstance *viper.Viper) error {
	viperInstance.SetConfigType("toml")
	if err := viperInstance.MergeConfig(bytes.NewReader(source.data)); err != nil {
		return fmt.Errorf("merge embedded config: %w", err)
	}

	return nil
}

func NewFileSource(_ string, filePath string, optional bool) Source {
	return fileSource{
		path:     filePath,
		optional: optional,
	}
}

func (source fileSource) Name() string {
	return source.path
}

func (source fileSource) Apply(viperInstance *viper.Viper) error {
	file, err := os.Open(filepath.Clean(source.path))
	if err != nil {
		if source.optional && errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("open file %s: %w", source.path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	viperInstance.SetConfigType("toml")
	if err := viperInstance.MergeConfig(file); err != nil {
		return fmt.Errorf("merge file %s: %w", source.path, err)
	}

	return nil
}

func NewEnvSource(_ string, bindings []EnvBinding) Source {
	return envSource{
		bindings: bindings,
	}
}

func (source envSource) Name() string {
	return "env"
}

func (source envSource) Apply(viperInstance *viper.Viper) error {
	if strings.TrimSpace(source.prefix) != "" {
		viperInstance.SetEnvPrefix(strings.ToUpper(source.prefix))
	}
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperInstance.AutomaticEnv()

	for _, key := range viperInstance.AllKeys() {
		if err := viperInstance.BindEnv(key); err != nil {
			return fmt.Errorf("bind env %s: %w", key, err)
		}
	}

	for _, binding := range source.bindings {
		if err := viperInstance.BindEnv(binding.Key, binding.EnvName); err != nil {
			return fmt.Errorf("bind env %s for %s: %w", binding.EnvName, binding.Key, err)
		}
	}

	return nil
}

func (source sourceError) Name() string {
	return source.name
}

func (source sourceError) Apply(_ *viper.Viper) error {
	return source.err
}
