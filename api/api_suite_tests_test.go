package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/auth"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

type ResultModel[P contracts.RequestParam, R any] map[string]interface{}

func (e ResultModel[P, R]) HasError() (yes bool) {
	if len(e) == 0 {
		return
	}
	_, yes = e["error"]
	return
}

func (e ResultModel[P, R]) Error() (er errors.ErrorModel[P]) {
	if e.HasError() {
		utils.StructCopy(e["error"], &er)
		return
	}
	return
}

func (e ResultModel[P, R]) Data() (out R) {
	utils.StructCopy(e["data"], &out)
	return
}

func (e ResultModel[P, R]) Parse(out interface{}) (err error) {
	return utils.StructCopy(e, out)
}

func createLoginRequest(signup auth.SignupRequest) auth.LoginRequest {
	return auth.LoginRequest{
		Username: signup.Username, Password: signup.Password,
	}
}

func createSignupRequest() auth.SignupRequest {
	return auth.SignupRequest{
		Username: fmt.Sprintf("user.%s@test.com", uuid.New().String()),
		Password: fmt.Sprintf("%d@Paswerd", time.Now().Unix())}
}

func registerUser[ID comparable, U auth.SessionUser[ID]](ctx context.Context, signupReq auth.SignupRequest, sessionService *auth.GenericSessionService[ID, U]) (out U){
	// Note: Context validation removed for now since it requires RequestContext
	var err ierrors.Error
	if out, err = sessionService.Signup(ctx); err != nil {
		panic(err)
	}
	return
}

func authenticateUser[ID comparable, U auth.SessionUser[ID]](ctx context.Context, signupReq auth.SignupRequest, sessionService *auth.GenericSessionService[ID, U]) (token auth.TokenResponse){
	// Simplified stub implementation
	return auth.TokenResponse{AccessToken: "stub-token"}
}

func inCtx[T contracts.RequestParam](ctx context.Context, param T) context.Context {
	return context.WithValue(ctx, param.ContextKey(), &param)
}