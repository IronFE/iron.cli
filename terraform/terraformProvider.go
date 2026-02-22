package terraform

import (
	"os"
	"os/exec"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/pkg/errors"
)

type ITerraformProvider interface {
	Terraform(workDir string) (*tfexec.Terraform, error)
}

type TerraformProvider struct {
}

func NewTerraformProvider() ITerraformProvider {
	return &TerraformProvider{}
}

func (p *TerraformProvider) Terraform(workDir string) (*tfexec.Terraform, error) {
	execPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, errors.Wrap(err, "could not find terraform executable in PATH")
	}
	if execPath == "" {
		return nil, errors.Errorf("could not find Terraform executable")
	}
	tf, err := tfexec.NewTerraform(workDir, execPath)
	if err != nil {
		return nil, errors.Wrap(err, "error finding terraform cli")
	}

	tf.SetStdout(os.Stdout)

	return tf, nil
}
