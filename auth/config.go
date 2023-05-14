package auth

import (
	"fmt"
	"strings"

	"github.com/kod2ulz/gostart/utils"
)

type Config struct {
	Driver             string
	UserPool           string
	ClientID           string
	ClientSecret       string
	AuthIssuerURL      string
	JwkRefreshInterval utils.Value
	PublicKeyURL       string
}

func Conf(prefix ...string) (conf *Config) {
	env := utils.Env.Helper(prefix...).OrDefault("AUTH")
	conf = &Config{
		Driver:             env.GetString("DRIVER", cognitoDriver),
		UserPool:           env.GetString("USER_POOL", ""),
		ClientID:           env.GetString("CLIENT_ID", ""),
		JwkRefreshInterval: env.Get("JWK_REFRESH_INTERVAL", "15m"),
		ClientSecret:       env.GetString("CLIENT_SECRET", ""),
		AuthIssuerURL:      env.GetString("ISSUER_URL", "https://auth.startup.io"),
		PublicKeyURL:       env.GetString("PUBLIK_KEY_URL", ""),
	}
	if conf.PublicKeyURL != "" {
		return
	}
	switch conf.Driver {
	case cognitoDriver:
		if conf.UserPool != "" && strings.Contains(conf.UserPool, "_") {
			region := strings.Split(conf.UserPool, "_")[0]
			conf.PublicKeyURL = fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, conf.UserPool)
		}
	}
	return
}