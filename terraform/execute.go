package terraform

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/IronFE/iron.cli/util"
	"github.com/IronFE/iron.cli/util/aws"
	"github.com/IronFE/iron.cli/util/config"
	"github.com/IronFE/iron.cli/util/git"
	"github.com/apex/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	_ "embed"
)

//go:embed config.tf.tmpl
var configTemplate string

type ITerraformExecution interface {
	Execute(action func(tf *tfexec.Terraform, options ExecutionOptions) error) error
}

type execution struct {
	provider ITerraformProvider

	accountAlias   string
	authProfile    string
	deploymentName string
	keepTemp       bool
	pathArg        string
	roleToAssume   string
	noRoleAssume   bool
	mfa            bool
	workDir        string
	logLevel       string
	variant        string
}

type ExecutionOptions struct {
	VariableFiles []string
}

type CliOptions struct {
	AuthProfile    string
	DebugLevel     string
	DeploymentName string
	KeepTempDir    bool
	Mfa            bool
	NoRoleAssume   bool
	RoleToAssume   string
	TargetAccount  string
	WorkDir        string
	Variant        string
}

func NewTerraformExecution(options *CliOptions) ITerraformExecution {
	return &execution{
		provider:       NewTerraformProvider(),
		deploymentName: options.DeploymentName,
		accountAlias:   options.TargetAccount,
		authProfile:    options.AuthProfile,
		mfa:            options.Mfa,
		noRoleAssume:   options.NoRoleAssume,
		keepTemp:       options.KeepTempDir,
		logLevel:       options.DebugLevel,
		pathArg:        options.WorkDir,
		roleToAssume:   options.RoleToAssume,
		variant:        options.Variant,
	}
}

func (e *execution) Execute(action func(tf *tfexec.Terraform, options ExecutionOptions) error) error {
	awsAbstraction, err := aws.NewAws(e.authProfile)
	if err != nil {
		return err
	}

	if e.workDir, err = util.GetWorkDirFromArg(e.pathArg); err != nil {
		return err
	}

	var access *aws.AwsAccountAccess

	if e.noRoleAssume {
		access, err = awsAbstraction.SessionToken(30 * time.Minute)
		if err != nil {
			return fmt.Errorf("failed to get session token for current user: %w", err)
		}
	} else {

		if e.mfa {
			access, err = awsAbstraction.AssumeRoleWithMfa(e.roleToAssume, e.accountAlias)
		} else {
			access, err = awsAbstraction.AssumeRole(e.roleToAssume, e.accountAlias)
		}

		if err != nil {
			return err
		}
	}

	cfg, err := readTerraformConfig(e.workDir)
	if err != nil {
		return err
	}

	var deploymentName string
	if e.deploymentName != "" {
		deploymentName = e.deploymentName
	} else {
		pathParts := strings.Split(e.workDir, "/")
		deploymentName = pathParts[len(pathParts)-1]
	}

	tags := map[string]string{
		"deployment": deploymentName,
	}

	gitSource, err := git.OriginRemote(e.workDir)
	if err == nil {
		tags["source"] = gitSource
	}

	for _, provider := range cfg.Providers {
		if provider.Name == "aws" {
			if provider.Tags != nil {
				for k, v := range tags {
					provider.Tags[k] = v
				}
			} else {
				provider.Tags = tags
			}
		}
	}

	if cfg.Backend.Type == "s3" {
		cfg.Backend.Config["key"] = deploymentName
		if _, exists := cfg.Backend.Config["bucket"]; !exists {
			cfg.Backend.Config["bucket"] = fmt.Sprintf("%s-tf-state", access.AccountId)
		}
	}

	return e.onWorkingCopy(access, cfg, func(credentials *aws.AwsAccountAccess, workDir string) error {
		tf, err := e.provider.Terraform(workDir)
		if err != nil {
			return err
		}

		userEnvs := lo.SliceToMap(os.Environ(), func(item string) (string, string) {
			splits := strings.SplitN(item, "=", 2)
			return splits[0], splits[1]
		})

		userEnvs["AWS_ACCESS_KEY_ID"] = credentials.AccessKeyId
		userEnvs["AWS_SECRET_ACCESS_KEY"] = credentials.SecretKey
		userEnvs["AWS_SESSION_TOKEN"] = credentials.SessionToken

		if err = tf.SetEnv(userEnvs); err != nil {
			return fmt.Errorf("failed to set environment variables for terraform: %w", err)
		}

		if e.logLevel != "" {
			if err = tf.SetLog(e.logLevel); err != nil {
				return fmt.Errorf("failed to set the terraform log level to %q: %w", e.logLevel, err)
			}
			if err = tf.SetLogPath(filepath.Join(workDir, "terraform.log")); err != nil {
				return fmt.Errorf("failed to set the terraform log path: %w", err)
			}
		}

		err = tf.Init(context.Background(), tfexec.Upgrade(true))
		if err != nil {
			return errors.Wrap(err, "error running terraform init")
		}

		variableFiles, err := e.variableFiles(workDir)
		if err != nil {
			return fmt.Errorf("the variant file can not be read: %w", err)
		}

		if err = action(tf, ExecutionOptions{VariableFiles: variableFiles}); err != nil {
			return err
		}

		return nil
	})
}

func (e *execution) onWorkingCopy(account *aws.AwsAccountAccess, cfg *config.TerraformConfig, action func(account *aws.AwsAccountAccess, workDir string) error) error {

	base := filepath.Join(e.workDir, "..")

	dest, err := os.MkdirTemp(base, ".tf")
	if err != nil {
		return errors.Wrap(err, "error while creating temp directory for terraform")
	}

	log.Infof("working on temp dir %s", dest)
	log.Infof("AWS account id: %s, account name: %s", account.AccountId, e.accountAlias)
	if err = util.CopyFolder(e.workDir, dest); err != nil {
		return fmt.Errorf("error while copying terraform directory: %w", err)
	}

	if err = e.addConfig(dest, cfg); err != nil {
		return err
	}

	actionErr := action(account, dest)

	lockFileName := ".terraform.lock.hcl"
	lockFilePath := path.Join(dest, lockFileName)
	lockFileExists, err := util.FileExists(lockFilePath)
	if err != nil {
		log.WithError(err).Warnf("terraform lock file existence could not be checked ")
	} else {
		if lockFileExists {
			if err = util.CopyFile(lockFilePath, path.Join(e.workDir, lockFileName)); err != nil {
				log.WithError(err).Warnf("terraform lock file could not be copied")
			}
		}
	}

	if e.keepTemp {
		return actionErr
	}

	if err = os.RemoveAll(dest); err != nil {
		return errors.Wrap(err, "failed to clean up temp directory for terraform")
	}

	return actionErr
}

func (e *execution) addConfig(workDir string, cfg *config.TerraformConfig) error {

	providersFile, err := os.Create(filepath.Join(workDir, "providers.tf"))
	if err != nil {
		return errors.Wrap(err, "failed to create providers.tf")
	}

	defer func() {
		_ = providersFile.Close()
	}()

	tmpl, err := template.New("terraformConfig").Parse(configTemplate)

	if err != nil {
		return err
	}
	err = tmpl.Execute(providersFile, cfg)
	return err
}

func (e *execution) variableFiles(workDir string) ([]string, error) {
	if e.variant == "" {
		return []string{}, nil
	}
	filePath := path.Join(workDir, fmt.Sprintf("variants/%s.tfvars", e.variant))
	_, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("the file %s does not exist or is not readable: %w", filePath, err)
	}
	return []string{filePath}, nil
}
