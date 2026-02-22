package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"gopkg.in/yaml.v3"
)

type provider struct {
	data *configFile
}

type IProvider interface {
	Profile(name string) (Profile, error)
	DefaultProfile() (Profile, error)
	Terraform() (TerraformConfig, error)
}

type configFile struct {
	Profiles  []Profile       `yaml:"profiles"`
	Terraform TerraformConfig `yaml:"terraform"`
}

type Profile struct {
	Name           string          `yaml:"name"`
	Default        bool            `yaml:"default"`
	AuthStrategy   string          `yaml:"authStrategy"`
	IdentityCenter *IdentityCenter `yaml:"identityCenter"`
	IAM            *IAM            `yaml:"iam"`
	DefaultRegion  string          `yaml:"defaultRegion"`
}

type IdentityCenter struct {
	StartUrl    string `yaml:"startUrl"`
	DefaultRole string `yaml:"defaultRole"`
	Region      string `yaml:"region"`
}

type IAM struct {
	ProfileName string `yaml:"profileName"`
	MfaSerial   string `yaml:"mfaSerial"`
}

func NewProfileProvider() IProvider {
	return &provider{}
}

func (p *provider) DefaultProfile() (Profile, error) {
	if err := p.initialize(); err != nil {
		return Profile{}, err
	}
	configFilePath := p.configFilePath()

	if len(p.data.Profiles) == 0 {
		return Profile{}, fmt.Errorf("no profiles configured at %s", configFilePath)
	}

	for _, profile := range p.data.Profiles {
		if profile.Default {
			return profile, nil
		}
	}
	defaultProfile := p.data.Profiles[0]
	log.Warnf("no default profile configured in %s. defaulting to the first: %s", configFilePath, defaultProfile.Name)
	return defaultProfile, nil
}

func (p *provider) Profile(name string) (Profile, error) {
	if err := p.initialize(); err != nil {
		return Profile{}, err
	}

	for _, profile := range p.data.Profiles {
		if profile.Name == name {
			return profile, nil
		}
	}

	return Profile{}, fmt.Errorf("could not find profile named %q", name)
}

func (p *provider) Terraform() (TerraformConfig, error) {
	if err := p.initialize(); err != nil {
		return TerraformConfig{}, err
	}
	return p.data.Terraform, nil
}

func (p *provider) initialize() error {
	if p.data == nil {
		cfg, err := p.readConfig()
		if err != nil {
			return err
		}
		p.data = &cfg
	}
	return nil
}

func (p *provider) readConfig() (configFile, error) {
	configFilePath := p.configFilePath()
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return configFile{}, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := configFile{}
	if err = yaml.Unmarshal(content, &cfg); err != nil {
		return configFile{}, fmt.Errorf("could not parse config file %q: %w", configFilePath, err)
	}
	return cfg, nil
}

func (p *provider) configFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.WithError(err).Warn("failed to get the home directory")
		home = os.Getenv("HOME")
	}
	baseFolder := filepath.Join(home, ".iron-cli")

	configFilePath := filepath.Join(baseFolder, "config.yaml")
	return configFilePath
}
