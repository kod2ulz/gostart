package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/kod2ulz/gostart/utils"
)

type TokenConfig struct {
	AccessTimeout  time.Duration
	RefreshTimeout time.Duration
	Issuer         string
	ClientID       string
	ClientSecret   string
	SigningKeySeed string
	SigningKey     []byte
	Audience       []string
}

func InitTokenConfig(prefix ...string) (out *TokenConfig) {
	env := utils.Env.Helper(prefix...).OrDefault("TOKEN")
	out = &TokenConfig{
		AccessTimeout:  env.Get("ACCESS_TIMEOUT", "60m").Duration(),
		Issuer:         env.Get("ISSUER", fmt.Sprintf("http://%s", utils.Env.GetHost())).String(),
		ClientID:       env.Get("CLIENT_ID", "TQcXMsCGc3RaMlHiUfiF").String(),
		ClientSecret:   env.Get("CLIENT_SECRET", "MHz7SszY1ujSFp9TFMNU").String(),
		SigningKeySeed: env.Get("SIGNING_KEY", "").String(),
		RefreshTimeout: env.Get("REFRESH_TIMEOUT", "24h").Duration(),
		Audience:       env.Get("AUDIENCE", "http://localhost,api_client").StringList(","),
	}
	if out.SigningKeySeed == "" {
		out.SigningKeySeed = fmt.Sprintf("%s%s%s", out.Issuer, out.ClientID, out.ClientSecret)
	}
	out.SigningKey = []byte(out.hashFunc(out.SigningKeySeed))
	return
}

func (s *TokenConfig) hashFunc(phrase string) string {
	mac := hmac.New(sha256.New, []byte(s.ClientSecret))
	mac.Write([]byte(phrase + s.ClientID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
