package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/services/auth"
	"github.com/kod2ulz/gostart/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

type ResultModel[P api.RequestParam, R any] map[string]interface{}

func (e ResultModel[P, R]) HasError() (yes bool) {
	if len(e) == 0 {
		return
	}
	_, yes = e["error"]
	return
}

func (e ResultModel[P, R]) Error() (er api.ErrorModel[P]) {
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
	ctx = inCtx(ctx, signupReq)
	var err api.Error
	if e := signupReq.Validate(ctx); e != nil {
		panic(e)
	} else if out, err = sessionService.Signup(inCtx(ctx, signupReq)); err != nil {
		panic(err)
	}
	return
}

func authenticateUser[ID comparable, U auth.SessionUser[ID]](ctx context.Context, signupReq auth.SignupRequest, sessionService *auth.GenericSessionService[ID, U]) (token auth.TokenResponse){
	var err api.Error
	loginReq := createLoginRequest(signupReq)
	if e := loginReq.Validate(inCtx(ctx, loginReq)); e != nil {
		panic(e)
	} else if token, err = sessionService.Login(inCtx(ctx, loginReq)); err != nil {
		panic(err)
	}
	return
}

func inCtx[T api.RequestParam](ctx context.Context, param T) context.Context {
	return context.WithValue(ctx, param.ContextKey(), &param)
}