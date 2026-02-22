package aws

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/pkg/browser"
	"gopkg.in/yaml.v3"
)

type fileCachedAuthProvider struct {
	startUrl string
	region   string
}

type authFile struct {
	SessionToken   string    `yaml:"sessionToken"`
	ExpirationDate time.Time `yaml:"expirationDate"`
}

func (p *fileCachedAuthProvider) Auth() (string, error) {

	token, err := p.readFromFile()
	if err != nil {
		log.WithError(err).Warn("could not get token from cache")

		output, err := p.createNewToken()
		if err != nil {
			return "", fmt.Errorf("failed to create new auth token: %w", err)
		}

		expirationDate := time.Now().Add(time.Duration(output.ExpiresIn) * time.Second)

		if err = p.writeToFile(*output.AccessToken, expirationDate); err != nil {
			log.WithError(err).Warn("failed to cache token in file")
		}

		return *output.AccessToken, nil

	}

	return token, err
}

func (p *fileCachedAuthProvider) createNewToken() (*ssooidc.CreateTokenOutput, error) {
	// code based on https://gist.github.com/ayubmalik/5b5b83b8153c0afdc1d31d5380001ff0
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithDefaultRegion(p.region))
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	// create SSO oidcClient client to trigger login flow
	oidcClient := ssooidc.NewFromConfig(cfg)

	// register your client which is triggering the login flow
	register, err := oidcClient.RegisterClient(context.Background(), &ssooidc.RegisterClientInput{
		ClientName: aws.String("sso-iron-cli"),
		ClientType: aws.String("public"),
	})

	if err != nil {
		return nil, err
	}

	// authorize your device using the client registration response
	deviceAuth, err := oidcClient.StartDeviceAuthorization(context.Background(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(p.startUrl),
	})
	if err != nil {
		return nil, err
	}

	url := aws.ToString(deviceAuth.VerificationUriComplete)
	log.Infof("If your browser is not opened automatically, please open link:\n%v\n", url)

	err = browser.OpenURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to open brower with url %q: %w", url, err)
	}

	for {
		output, err := oidcClient.CreateToken(context.Background(), &ssooidc.CreateTokenInput{
			ClientId:     register.ClientId,
			ClientSecret: register.ClientSecret,
			DeviceCode:   deviceAuth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err != nil {
			isPending := strings.Contains(err.Error(), "AuthorizationPendingException:")
			if isPending {
				log.Debug("Authorization pending...")
				time.Sleep(time.Duration(deviceAuth.Interval) * time.Second)
				continue
			}
		}

		return output, nil
	}
}

func (p *fileCachedAuthProvider) writeToFile(token string, expirationDate time.Time) error {
	filePath, err := p.filePath()
	if err != nil {
		return err
	}

	data := authFile{SessionToken: token, ExpirationDate: expirationDate}
	bytes, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("failed to marshal auth file data: %w", err)
	}

	if err = os.WriteFile(filePath, bytes, 0600); err != nil {
		return fmt.Errorf("failed to store auth data in file: %w", err)
	}

	return nil
}

func (p *fileCachedAuthProvider) readFromFile() (string, error) {

	authFilePath, err := p.filePath()
	if err != nil {
		return "", err
	}

	_, err = os.Stat(authFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("the auth file does not exist")
	}

	content, err := os.ReadFile(authFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read auth file: %w", err)
	}

	auth := authFile{}
	if err = yaml.Unmarshal(content, &auth); err != nil {
		return "", fmt.Errorf("could not parse auth file %q: %w", authFilePath, err)
	}

	if auth.ExpirationDate.After(time.Now().Add(2 * time.Second)) {
		return auth.SessionToken, nil
	}

	return "", fmt.Errorf("token is expired")
}

func (p *fileCachedAuthProvider) filePath() (string, error) {
	startUrl, err := url.Parse(p.startUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse start url %q: %w", p.startUrl, err)
	}

	home := os.Getenv("HOME")
	baseFolder := filepath.Join(home, ".iron-cli")

	return filepath.Join(baseFolder, fmt.Sprintf("%s.yaml", startUrl.Host)), nil
}
