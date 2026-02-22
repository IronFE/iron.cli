package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/IronFE/iron.cli/util/aws"
	"github.com/apex/log"
	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

type ssmSessionOptions struct {
	authProfile      string
	accountName      string
	credsOnly        bool
	instanceNameOrId string
	role             string
	region           string
	portForwarding   string
}

func NewSsmSessionCommand() *cobra.Command {
	options := ssmSessionOptions{}
	cmd := &cobra.Command{
		Use:   "ssm-session",
		Short: "Creates an SSM-based ssh session to an EC2 instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.instanceNameOrId = args[0]
			return startSession(options)
		},
	}

	cmd.Flags().StringVarP(&options.authProfile, "profile", "p", "", "The AWS credentials profile to use")
	cmd.Flags().StringVarP(&options.accountName, "account", "a", "", "sets the account name")
	cmd.Flags().StringVarP(&options.role, "role", "r", "", "sets the role to assume")
	cmd.Flags().StringVar(&options.region, "region", "", "sets the region to use")
	cmd.Flags().StringVar(&options.portForwarding, "port", "", "Allows to forward exactly one port. Use <local_port>:<remote_port>")

	return cmd
}

func startSession(options ssmSessionOptions) error {
	var err error
	var awsAbstraction aws.IAws
	awsAbstraction, err = aws.NewAws(options.authProfile)
	if err != nil {
		return err
	}

	creds, err := awsAbstraction.AssumeRole(options.role, options.accountName)
	if err != nil {
		return fmt.Errorf("failed to assume role %q in %q: %w", options.role, options.accountName, err)
	}

	region := options.region
	if region == "" {
		region = awsAbstraction.Region()
	}
	args := []string{
		"ssm", "start-session",
		"--target", instanceId(options.instanceNameOrId, creds, region),
	}
	if options.portForwarding != "" {
		args = append(args, []string{"--document-name", "AWS-StartPortForwardingSession"}...)

		splitting := strings.Split(options.portForwarding, ":")
		if len(splitting) != 2 {
			return fmt.Errorf("invalid port forwarding format: %s", options.portForwarding)
		}

		localPort := []string{splitting[0]}
		remotePort := []string{splitting[1]}

		document := struct {
			PortNumber      []string `json:"portNumber"`
			LocalPortNumber []string `json:"localPortNumber"`
		}{
			PortNumber:      remotePort,
			LocalPortNumber: localPort,
		}
		rawJson, err := json.Marshal(&document)
		if err != nil {
			return fmt.Errorf("failed to marshal port forwarding document: %w", err)
		}

		args = append(args, []string{"--parameters", string(rawJson)}...)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "aws", args...)

	envs := os.Environ()

	allEnvs := make([]string, len(envs)+3)
	copy(allEnvs, envs)

	allEnvs[len(envs)] = fmt.Sprintf("%s=%s", "AWS_ACCESS_KEY_ID", creds.AccessKeyId)
	allEnvs[len(envs)+1] = fmt.Sprintf("%s=%s", "AWS_SECRET_ACCESS_KEY", creds.SecretKey)
	allEnvs[len(envs)+2] = fmt.Sprintf("%s=%s", "AWS_SESSION_TOKEN", creds.SessionToken)

	cmd.Env = allEnvs

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	killed := false
	go func() {

		var lastHitTime = time.Now()
		var hitCount = 0
		for {
			select {
			case <-signalChan:
				if time.Now().After(lastHitTime.Add(2 * time.Second)) {
					hitCount = 1
					lastHitTime = time.Now()
					continue
				}

				if hitCount == 0 {
					lastHitTime = time.Now()
					hitCount = 1
					continue
				}

				killed = true
				cancel()
				return
			case <-ctx.Done():
				fmt.Println("Process exited on its own")

				return
			}
		}
	}()

	err = cmd.Run()

	if killed {
		os.Exit(0)
		return nil
	}

	return err
}

func instanceId(nameOrInstanceId string, creds *aws.AwsAccountAccess, region string) string {
	credsProvider := credentials.NewStaticCredentialsProvider(creds.AccessKeyId, creds.SecretKey, creds.SessionToken)
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region), config.WithCredentialsProvider(credsProvider))
	if err != nil {
		log.WithError(err).Warn("failed to load aws config")
		return nameOrInstanceId
	}

	ec2Client := ec2.NewFromConfig(cfg)
	input := ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: awsSdk.String("tag:Name"), Values: []string{nameOrInstanceId}},
			{Name: awsSdk.String("instance-state-name"), Values: []string{"running"}},
		},
	}

	output, err := ec2Client.DescribeInstances(context.Background(), &input)
	if err != nil {
		log.WithError(err).Warn("failed to describe ec2 instances")
		return nameOrInstanceId
	}

	var instanceId string = nameOrInstanceId
	var instanceFound = false
outer:
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceId = *instance.InstanceId
			instanceFound = true
			break outer

		}
	}
	if !instanceFound {
		log.Warn("No matching ec2 instance found; using input as instance id")
	}

	return instanceId
}
