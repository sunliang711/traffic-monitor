package config

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Source interface {
	Name() string
	Load(viperInstance *viper.Viper) error
}

type EmbeddedFileSource struct {
	name     string
	fileName string
	fileSet  embed.FS
}

type FileSource struct {
	name      string
	filePath  string
	optional  bool
}

type EnvBinding struct {
	Key     string
	EnvName string
}

type EnvSource struct {
	name     string
	bindings []EnvBinding
}

func NewEmbeddedFileSource(name string, fileSet embed.FS, fileName string) EmbeddedFileSource {
	return EmbeddedFileSource{
		name:     name,
		fileName: fileName,
		fileSet:  fileSet,
	}
}

func (source EmbeddedFileSource) Name() string {
	return source.name
}

func (source EmbeddedFileSource) Load(viperInstance *viper.Viper) error {
	fileContent, err := source.fileSet.ReadFile(source.fileName)
	if err != nil {
		return fmt.Errorf("read embedded file: %w", err)
	}

	if err := viperInstance.MergeConfig(bytes.NewReader(fileContent)); err != nil {
		return fmt.Errorf("merge embedded config: %w", err)
	}

	return nil
}

func NewFileSource(name string, filePath string, optional bool) FileSource {
	return FileSource{
		name:     name,
		filePath: filePath,
		optional: optional,
	}
}

func (source FileSource) Name() string {
	return source.name
}

func (source FileSource) Load(viperInstance *viper.Viper) error {
	file, err := os.Open(filepath.Clean(source.filePath))
	if err != nil {
		if source.optional && os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("open file %s: %w", source.filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	if err := mergeFromReader(viperInstance, file); err != nil {
		return fmt.Errorf("merge file %s: %w", source.filePath, err)
	}

	return nil
}

func NewEnvSource(name string, bindings []EnvBinding) EnvSource {
	return EnvSource{
		name:     name,
		bindings: bindings,
	}
}

func (source EnvSource) Name() string {
	return source.name
}

func (source EnvSource) Load(viperInstance *viper.Viper) error {
	viperInstance.AutomaticEnv()

	for _, binding := range source.bindings {
		if err := viperInstance.BindEnv(binding.Key, binding.EnvName); err != nil {
			return fmt.Errorf("bind env %s: %w", binding.EnvName, err)
		}
	}

	return nil
}

func mergeFromReader(viperInstance *viper.Viper, reader io.Reader) error {
	if err := viperInstance.MergeConfig(reader); err != nil {
		return fmt.Errorf("merge config: %w", err)
	}

	return nil
}
