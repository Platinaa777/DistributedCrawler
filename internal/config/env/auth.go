package env

import (
	"distributed-crawler/internal/config"
	"fmt"
	"os"
)

const (
	jwtSecretEnvName       = "JWT_SECRET"
	accessTokenTTLEnvName  = "ACCESS_TOKEN_TTL"
	refreshTokenTTLEnvName = "REFRESH_TOKEN_TTL"
	jwtIssuerEnvName       = "JWT_ISSUER"
	jwtAudienceEnvName     = "JWT_AUDIENCE"
	defaultUserEmailEnv    = "DEFAULT_USER_EMAIL"
	defaultUserPasswordEnv = "DEFAULT_USER_PWD"

	defaultAccessTokenTTL  = "15m"
	defaultRefreshTokenTTL = "720h" // 30 days
	defaultIssuer          = "distributed-crawler"
	defaultAudience        = "api"
)

type authConfig struct {
	jwtSecret       string
	accessTokenTTL  string
	refreshTokenTTL string
	issuer          string
	audience        string
	defaultEmail    string
	defaultPassword string
}

func NewAuthConfig() (config.AuthConfig, error) {
	jwtSecret := os.Getenv(jwtSecretEnvName)
	if len(jwtSecret) == 0 {
		return nil, fmt.Errorf("%s environment variable is required", jwtSecretEnvName)
	}

	accessTokenTTL := os.Getenv(accessTokenTTLEnvName)
	if len(accessTokenTTL) == 0 {
		accessTokenTTL = defaultAccessTokenTTL
	}

	refreshTokenTTL := os.Getenv(refreshTokenTTLEnvName)
	if len(refreshTokenTTL) == 0 {
		refreshTokenTTL = defaultRefreshTokenTTL
	}

	issuer := os.Getenv(jwtIssuerEnvName)
	if len(issuer) == 0 {
		issuer = defaultIssuer
	}

	audience := os.Getenv(jwtAudienceEnvName)
	if len(audience) == 0 {
		audience = defaultAudience
	}

	defaultEmail := os.Getenv(defaultUserEmailEnv)
	if len(defaultEmail) == 0 {
		return nil, fmt.Errorf("%s environment variable is required", defaultUserEmailEnv)
	}

	defaultPassword := os.Getenv(defaultUserPasswordEnv)
	if len(defaultPassword) == 0 {
		return nil, fmt.Errorf("%s environment variable is required", defaultUserPasswordEnv)
	}

	return &authConfig{
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		issuer:          issuer,
		audience:        audience,
		defaultEmail:    defaultEmail,
		defaultPassword: defaultPassword,
	}, nil
}

func (cfg *authConfig) JWTSecret() string {
	return cfg.jwtSecret
}

func (cfg *authConfig) AccessTokenTTL() string {
	return cfg.accessTokenTTL
}

func (cfg *authConfig) RefreshTokenTTL() string {
	return cfg.refreshTokenTTL
}

func (cfg *authConfig) Issuer() string {
	return cfg.issuer
}

func (cfg *authConfig) Audience() string {
	return cfg.audience
}

func (cfg *authConfig) DefaultUserEmail() string {
	return cfg.defaultEmail
}

func (cfg *authConfig) DefaultUserPassword() string {
	return cfg.defaultPassword
}
