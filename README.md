# iron-cli

The Iron-CLI provides an easy way to provision resources on AWS using Terraform in a multi-account environment using
AWS Organizations and AWS IAM Identity Center. It manages authentication, account-, state- and variable handling and 
allows you as a developer to focus on the development of Terraform code.

It has a few utility commands available as well to interact quickly with AWS.

## Commands
### Deploy
```shell
iron deploy --account dev --confirm .
```
Runs a Terraform deployment in the current folder against the account named `dev` and waits for the users approval
after the plan was outputted (`--confirm`)

````shell
iron deploy --profile other_corp --account testing --role SuperAdmin --variant blue .
````
Runs a Terraform deployment in the current folder against the account `testing` of the organization you named 
`other_corp` assuming the role `SuperAdmin` with the .tfvars file named `blue.tfvars` in the variants folder of the 
deployment. This will not wait for the users approval but apply the changes right away.


#### destroy
```shell
iron destroy --account dev --confirm .
```
Runs Terraform with the destroy option against the `dev` account and wait for the users' approval.


#### plan 
```shell
iron plan --acount dev .
```
Runs Terraform plan against the `dev` account.

#### authorize
```shell
iron authorize --account dev -- aws ec2 describe-addresses
```
Executes any command with the AWS permissions of the default role you defined (in the config) in the account `dev`.

#### ssm-session
```shell
iron ssm-session --account dev web
```
Starts an SSH session using AWS Systems Manager on the EC2 instance named `web` in the account `dev`.

#### ecr
```shell
iron ecr --account dev login
```
Performs a `docker login` into the AWS ECR registry in the account `dev`

## Installation
Create the file `~/.iron-cli/config.yaml` with the following content
```yaml
profiles:
  - name: <your preferred profile name>
    defaultRegion: <your default AWS region>
    authStrategy: identityCenter
    identityCenter:
      startUrl: https://<your IAM Identity Center domain>.awsapps.com/start
      defaultRole: <the default Identity Center role you want to work with> 
      region: <the primary AWS region for AWS Identity Center; if empty, the defaultRegion from the profile will be used>

terraform:
  terraform_version: ">= 1.2.0"
  providers:
    - name: "aws"
      source: "hashicorp/aws"
      config:
        region: <any AWS region you want to use as a default>
  backend:
    type: s3
    config:
      region: <the AWS region where your S3 Bucket with the Terraform states will be>
```

## Terraform
All Terraform states will be stored in a S3 backend. The bucket in the account of the deployment must be named 
<AWS-Account-ID>-tf-state. During Terraform operations, a temporary folder will be created beneath the folder of your 
deployment. This folder contains a copy of your code. During the deployment you can make further code changes, without
influencing any running Terraform operation. The temporary folder also contains an additional file named "providers.tf".
The file is auto generated and contains the basic Terraform configuration, as defined in the global config.yaml. 

You can override any global settings with a git repository wide config in the file "tf/config.yaml"
or a file "config.yaml" directly in the folder with the Terraform code. The file must be structured as the 
terraform-section in the global config.yaml mentioned above (but without the terraform-section on root-level). 
All available files will be merged together in the following order:
merge(merge(config.yaml, <git>/tf/config.yaml), <git>/<path-to-terraform>/config.yaml)
The config.yaml in your deployment folder will "win" over the config defined anywhere else.
