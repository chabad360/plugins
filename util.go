package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"path/filepath"
)

func walkZipHashes(zipHashes map[string]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".zip" {
			return fmt.Errorf("%w: File %v is not a zip file", ErrLoading, path)
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		hash := sha256.New()
		if _, err := io.Copy(hash, f); err != nil {
			return err
		}
		zipHashes[hex.EncodeToString(hash.Sum(nil))] = path

		return nil
	}
}

func walkPluginHashes(pluginHashes map[string]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) != "pluigin.yml" {
			return filepath.SkipDir
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		var c []byte
		if _, err := f.Read(c); err != nil {
			return err
		}

		var config PluginConfig
		if err := yaml.Unmarshal(c, &config); err != nil {
			return err
		}

		if config.Local {
			hash := sha256.New()
			if _, err := io.Copy(hash, f); err != nil {
				return err
			}
			pluginHashes[fmt.Sprintf("local-%x", hash.Sum(nil))] = path
		} else {
			pluginHashes[config.Hash] = path
		}

		return nil
	}
}
