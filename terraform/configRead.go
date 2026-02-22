package terraform

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/IronFE/iron.cli/util/config"
	"github.com/IronFE/iron.cli/util/git"
	"github.com/apex/log"
	"gopkg.in/yaml.v3"
)

func readTerraformConfig(workDir string) (*config.TerraformConfig, error) {
	profileProvider := config.NewProfileProvider()
	cfg, err := profileProvider.Terraform()
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform defaults from config file")
	}

	gitRoot, err := git.GetRootDir(workDir)
	if err != nil {
		log.WithError(err).Warn("failed to get root of git")
	} else {
		searchPath := filepath.Join(gitRoot, "tf/config.yaml")
		cfg, err = mergeWithFileConfig(searchPath, cfg)
		if err != nil {
			return nil, err
		}
	}

	searchPath := filepath.Join(workDir, "config.yaml")
	cfg, err = mergeWithFileConfig(searchPath, cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func mergeWithFileConfig(searchPath string, cfg config.TerraformConfig) (config.TerraformConfig, error) {
	if _, err := os.Stat(searchPath); err == nil {
		data, err := os.ReadFile(searchPath)
		if err == nil {
			repoConfig := config.TerraformConfig{}
			if err = yaml.Unmarshal(data, &repoConfig); err == nil {
				cfg.Merge(repoConfig)
			} else {
				return config.TerraformConfig{}, fmt.Errorf("failed to parse %s", searchPath)
			}
		} else {
			log.WithError(err).Warnf("failed to read %s", searchPath)
		}
	}
	return cfg, nil
}
