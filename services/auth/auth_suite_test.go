package auth_test

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/services/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var log *logr.Logger

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Services:auth Suite")
}

func checkResponse[T any](recorder *httptest.ResponseRecorder, res *api.Response[T]) {
	Expect(recorder.Code).To(Equal(http.StatusOK))
	Expect(json.NewDecoder(recorder.Body).Decode(res)).To(BeNil())
	Expect(res.Success).To(BeTrue())
	Expect(res.Error).To(BeNil())
	Expect(res.Meta).To(BeNil())
	Expect((time.Now().Unix() - res.Timestamp) < 10).To(BeTrue())
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