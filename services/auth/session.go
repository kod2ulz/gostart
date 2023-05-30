package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/logr"
	"github.com/pkg/errors"
)

const (
	TokenTypeBearer = "Bearer"
	JwtLeewayWindow = 5 * time.Second
)

type SessionUser[ID comparable] interface {
	GetID() ID
	GetUsername() string
	GetPassword() string
	IsDisabled() bool
}

type SessionStore[ID comparable, U SessionUser[ID]] interface {
	GetUserWithID(context.Context, interface{}) (U, error)
	GetUserWithUsername(context.Context, string) (U, error)
	CreateUser(context.Context, SignupRequest) (U, error)
	Clear()
}

type ServiceInitFunc[ID comparable, U SessionUser[ID]] func(s *GenericSessionService[ID, U])

func WithTokenConfig[ID comparable, U SessionUser[ID]](conf *TokenConfig) ServiceInitFunc[ID, U] {
	return func(s *GenericSessionService[ID, U]) { s.tokenConf = conf }
}

func SessionService[ID comparable, U SessionUser[ID]](log *logr.Logger, store SessionStore[ID, U], opts ...ServiceInitFunc[ID, U]) (out *GenericSessionService[ID, U]) {
	if store == nil {
		if log != nil {
			log.Fatal("SessionService: store is nil")
		} else {
			panic("SessionService: store is nil")
		}
	}
	out = &GenericSessionService[ID, U]{
		db: store, log: log,
		tokenConf: InitTokenConfig("SESSION_SERVICE_TOKEN"),
	}
	for i := range opts {
		opts[i](out)
	}
	return
}

var _ api.SessionService[User, TokenResponse] = (*GenericSessionService[uuid.UUID, User])(nil)

type GenericSessionService[ID comparable, U SessionUser[ID]] struct {
	db        SessionStore[ID, U]
	log       *logr.Logger
	tokenConf *TokenConfig
}

func (s *GenericSessionService[ID, U]) Auther() gin.HandlerFunc {
	return api.WithUser[VerifyTokenRequest, User, TokenResponse](s)
}

func (s *GenericSessionService[ID, U]) API(router *gin.RouterGroup) {
	router.
		POST("/login", api.ParamHandlerWithResponse[LoginRequest](s.Login)).
		POST("/verify", api.ParamHandlerWithResponse[VerifyTokenRequest](s.Verify)).
		POST("/refresh", api.ParamHandlerWithResponse[RefreshRequest](s.Refresh))
}

func (s *GenericSessionService[ID, U]) Signup(ctx context.Context) (out U, err api.Error) {
	var e error
	var params SignupRequest
	if e = params.LoadFromContext(ctx, &params); e != nil {
		return out, api.RequestLoadError[SignupRequest](errors.Wrap(e, "failed to load params"))
	} else if out, e = s.db.GetUserWithUsername(ctx, params.Username); e != nil && !errors.Is(e, ErrUserNotFound) {
		return out, api.GeneralError[U](e)
	}
	if out.GetUsername() == params.Username {
		return out, api.GeneralError[U](ErrUsernameTaken).WithErrorCode(StatusErrorCreation)
	} else if out, e = s.db.CreateUser(ctx, params.WithHash(s.tokenConf.hashFunc)); e != nil {
		return out, api.GeneralError[U](errors.Wrap(e, "failed to create user"))
	}
	return
}

func (s *GenericSessionService[ID, U]) Login(ctx context.Context) (out TokenResponse, err api.Error) {
	var e error
	var user SessionUser[ID]
	var params LoginRequest
	if e = params.LoadFromContext(ctx, &params); e != nil {
		return out, api.RequestLoadError[LoginRequest](errors.Wrap(e, "failed to load login params"))
	} else if user, e = s.db.GetUserWithUsername(ctx, params.Username); e != nil {
		return out, api.ServiceErrorUnauthorised(ErrLoginInvalid)
	} else if !params.verify(user, s.tokenConf.hashFunc) {
		return out, api.ServiceErrorUnauthorised(ErrLoginInvalid)
	} else if user.IsDisabled() {
		return out, api.ServiceErrorUnauthorised(ErrLoginDisabled)
	}
	out = TokenResponse{
		ExpiresIn: int(s.tokenConf.AccessTimeout.Seconds()),
		TokenType: TokenTypeBearer,
	}
	if out.AccessToken, e = s.generateToken(user, s.tokenConf.AccessTimeout); e != nil {
		return out, ServiceErrorGeneratingToken(e)
	} else if out.RefreshToken, e = s.generateToken(user, s.tokenConf.RefreshTimeout); e != nil {
		return out, ServiceErrorGeneratingToken(e)
	}
	return
}

func (s *GenericSessionService[ID, U]) Verify(ctx context.Context) (out User, err api.Error) {
	var e error
	var claims *Claims
	var params VerifyTokenRequest
	if params, e = api.ParamsFromContext[VerifyTokenRequest](ctx); e != nil {
		return out, api.RequestLoadError[VerifyTokenRequest](errors.Wrap(e, "failed to load params"))
	} else if claims, e = s.validateToken(params.Token); e != nil {
		return out, api.ServiceErrorUnauthorised(e)
	}
	return *claims.AuthUser(), nil
}

func (s *GenericSessionService[ID, U]) Refresh(ctx context.Context) (out TokenResponse, err api.Error) {
	var e error
	var claims *Claims
	var user SessionUser[ID]
	var params RefreshRequest

	if e = params.LoadFromContext(ctx, &params); e != nil {
		return out, api.RequestLoadError[RefreshRequest](errors.Wrap(e, "failed to load token refresh params"))
	} else if claims, e = s.validateToken(params.RefreshToken); e != nil {
		return out, api.ServiceErrorUnauthorised(e)
	} else if user, e = s.db.GetUserWithID(ctx, claims.Subject); e != nil {
		return out, api.ServiceErrorUnauthorised(errors.Wrap(e, "invalid user"))
	}

	out = TokenResponse{
		RefreshToken: params.RefreshToken,
		ExpiresIn:    int(s.tokenConf.AccessTimeout.Seconds()),
		TokenType:    TokenTypeBearer,
	}
	if out.AccessToken, e = s.generateToken(user, s.tokenConf.AccessTimeout); e != nil {
		return out, ServiceErrorGeneratingToken(e)
	}
	return
}

func (s *GenericSessionService[ID, U]) generateToken(user SessionUser[ID], validity time.Duration) (out string, err error) {
	claims := Claims{
		user.GetUsername(),
		s.tokenConf.ClientID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(validity)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.tokenConf.Issuer,
			Subject:   fmt.Sprint(user.GetID()),
			ID:        uuid.NewString(),
			Audience:  s.tokenConf.Audience,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.tokenConf.SigningKey)
}

func (s *GenericSessionService[ID, U]) validateToken(tokenString string) (claims *Claims, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.tokenConf.SigningKey, nil
	}, jwt.WithLeeway(JwtLeewayWindow))
	var ok bool
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse token")
	} else if claims, ok = token.Claims.(*Claims); !ok || !token.Valid {
		return nil, ErrTokenValidation
	} else if err = claims.Validate(s.tokenConf.Issuer, s.tokenConf.ClientID); err != nil {
		return nil, err
	}

	return
}
