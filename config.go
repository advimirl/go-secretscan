package main

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

type Config struct {
	AccessTokens                 []AccessToken `yaml:"access_tokens"`
	BlacklistedStrings           []string      `yaml:"blacklisted_strings"`
	BlacklistedExtensions        []string      `yaml:"blacklisted_extensions"`
	BlacklistedFilenames         []string      `yaml:"blacklisted_filenames"`
	BlacklistedPaths             []string      `yaml:"blacklisted_paths"`
	BlacklistedEntropyExtensions []string      `yaml:"blacklisted_entropy_extensions"`
	BlacklistedProjectNames      []string      `yaml:"blacklisted_project_names"`
	Path                         string        `yaml:"-"`
}

type AccessToken struct {
	Token      string `yaml:"token"`
	URL        string `yaml:"base_url"`
	WorkerType string `yaml:"worker_type"`
}

func ParseConfig(options *Options) (*Config, error) {
	config := &Config{}
	data, err := ioutil.ReadFile(options.ConfigPath)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return config, err
	}

	if len(config.AccessTokens) < 1 {
		return config, errors.New("you need to provide at least one Access Token")
	}

	var ok = false

	for i := range config.AccessTokens {
		config.AccessTokens[i].Token = os.ExpandEnv(config.AccessTokens[i].Token)
		tk := config.AccessTokens[i].Token
		if strings.HasPrefix(tk, "file://") {
			config.AccessTokens[i].Token = readTokenFromFile(tk)
		}
		ok = len(config.AccessTokens[i].Token) != 0 && len(config.AccessTokens[i].URL) != 0
	}

	if !ok {
		return config, errors.New("you need to provide at least one Access Token")
	}

	config.Path = options.ConfigPath
	return config, nil
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = Config{}
	type plain Config

	err := unmarshal((*plain)(c))

	if err != nil {
		return err
	}

	return nil
}
