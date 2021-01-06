package plugins

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func walkZipHashes(zipHashes map[string]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".zip" {
			return nil
			// fmt.Errorf("walkZipHashes: %w: %v", ErrLoading, path)
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

		fmt.Println(hash.Sum(nil))

		return nil
	}
}

func walkPluginHashes(pluginHashes map[string]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) != "plugin.yml" {
			//return filepath.SkipDir
			return nil
		}

		c, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		var config PluginConfig
		if err := yaml.Unmarshal(c, &config); err != nil {
			return err
		}

		if config.Local {
			hash := sha256.New()
			if _, err := io.Copy(hash, bytes.NewReader(c)); err != nil {
				return err
			}
			pluginHashes[fmt.Sprintf("local-%x", hash.Sum(nil))] = path
		} else {
			pluginHashes[config.Hash] = path
		}

		return nil
	}
}
