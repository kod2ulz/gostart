package auth

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/kod2ulz/gostart/logr"
)

const (
	cognitoDriver = "cognito"
)

func Client(log *logr.Logger, ctx context.Context, conf *Config) *CognitoClient {
	awCfg, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		log.WithError(err).Fatal("failed to load default AWS config")
	} else if conf.Driver != cognitoDriver {
		log.Fatalf("unsupported auth provider %s", conf.Driver)
	}

	return &CognitoClient{
		AppClientID:     conf.ClientID,
		AppClientSecret: conf.ClientSecret,
		Client:          cognito.NewFromConfig(awCfg),
		IssuerUrl:       conf.AuthIssuerURL,
	}
}

type CognitoClient struct {
	*cognito.Client
	AppClientID     string
	AppClientSecret string
	IssuerUrl       string
}
