package auth

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/utils"
)

type Config struct {
	Driver             string
	UserPool           string
	ClientID           string
	ClientSecret       string
	AuthIssuerURL      string
	CountryID          uuid.UUID
	Country            string
	JwkRefreshInterval utils.Value
}

func Conf(prefix ...string) (conf *Config) {
	env := utils.Env.Helper(prefix...).OrDefault("AUTH")
	return &Config{
		Driver:             env.GetString("DRIVER", cognitoDriver),
		UserPool:           env.GetString("USER_POOL", ""),
		ClientID:           env.GetString("CLIENT_ID", ""),
		JwkRefreshInterval: env.Get("JWK_REFRESH_INTERVAL", "15m"),
		ClientSecret:       env.GetString("CLIENT_SECRET", ""),
		Country:            env.GetString("COUNTRY", "Uganda"),
		AuthIssuerURL:      env.GetString("ISSUER_URL", "https://auth.startup.io"),
	}
}

type userPool string

func (u userPool) region() string {
	if u == "" || !strings.Contains(string(u), "_") {
		return ""
	}
	return strings.Split(string(u), "_")[0]
}

func (u userPool) publicKeyUrl() string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", u.region(), u)
}
