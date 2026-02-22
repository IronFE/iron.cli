package config

type TerraformConfig struct {
	Providers        []*Provider
	Backend          Backend
	TerraformVersion string `yaml:"terraform_version"`
}

type Provider struct {
	Name   string
	Source string
	Tags   map[string]string
	Config map[string]string
}

type Backend struct {
	Type   string
	Config map[string]string
}

func (c *TerraformConfig) Merge(other TerraformConfig) {
	if other.TerraformVersion != "" {
		c.TerraformVersion = other.TerraformVersion
	}

	if other.Backend.Type != "" {
		c.Backend.Type = other.Backend.Type
		c.Backend.Config = other.Backend.Config
	}

	for _, provider := range other.Providers {
		found := false
		for _, existing := range c.Providers {
			if existing.Name == provider.Name {
				found = true
				if provider.Source != "" {
					existing.Source = provider.Source
				}
				for k, v := range provider.Config {
					if existing.Config == nil {
						existing.Config = make(map[string]string)
					}
					existing.Config[k] = v
				}
				for k, v := range provider.Tags {
					if existing.Tags == nil {
						existing.Tags = make(map[string]string)
					}
					existing.Tags[k] = v
				}
			}
		}
		if !found {
			c.Providers = append(c.Providers, provider)
		}
	}
}
