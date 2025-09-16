package api_test

import (
	"context"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/ginadapter"
	"github.com/kod2ulz/gostart/auth"
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/contracts"
	gerrors "github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/utils"
)

// Shared test types and functions
type Book struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Author    string     `json:"author"`
	Pages     int        `json:"pages"`
	CreatedBy *uuid.UUID `json:"createdBy,omitempty"`
}

type CreateBookRequest struct {
	ID     *uuid.UUID `json:"id,omitempty"`
	User   auth.User   `json:"-"`
	Name   string     `json:"name"   validate:"required"`
	Author string     `json:"author" validate:"required"`
	Pages  int        `json:"pages"  validate:"required,gt=200"`
	api.RequestModal[CreateBookRequest]
}

func (r CreateBookRequest) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	var out CreateBookRequest
	if loadErr := out.LoadFromJsonBody(ctx, &out); loadErr != nil {
		return param, gerrors.Errorf("failed to load request: %v", loadErr)
	}
	out.User, _ = auth.GetUser(ctx.Context()) // ignoring error because some tests won't need r.User
	ctx.(*ginadapter.GinRequestContext).Set(out.ContextKey(), out)
	return out, nil
}

func (r *CreateBookRequest) book(id uuid.UUID) (out *Book) {
	out = &Book{ID: id, Name: r.Name, Author: r.Author, Pages: r.Pages}
	if r.User != nil {
		out.CreatedBy = utils.PointerTo(r.User.ID())
	}
	return
}

type _bookService struct {
	data collections.Map[uuid.UUID, *Book]
}

func bookService() *_bookService {
	return &_bookService{make(collections.Map[uuid.UUID, *Book])}
}

func (s *_bookService) clear() {
	if len(s.data) == 0 {
		return
	}
	for k := range s.data {
		delete(s.data, k)
	}
}

func (s *_bookService) setRoutes(router *gin.RouterGroup, middleware ...gin.HandlerFunc) {
	router.Use(middleware...).
		POST("", api.ParamHandlerWithResponse[CreateBookRequest](s.createBook)).
		GET("", api.ParamHandlerWithListResponse[ListBooksRequest](s.listBooks)).
		GET("/:id", api.ParamHandlerWithResponse[DetailedBookRequest](s.getBookByID))
}

func (s *_bookService) seed(size int, user auth.User) (out []*Book, err error) {
	var creatorId *uuid.UUID
	if size < 1 {
		return out, gerrors.Errorf("seed size is required and cannot be 0")
	} else if user != nil {
		creatorId = utils.PointerTo(user.ID())
	}
	out = make([]*Book, size)
	for i := 0; i < size; i++ {
		id := uuid.New()
		s.data[id] = &Book{
			ID:        id,
			Name:      utils.String.Random(20),
			Author:    utils.String.Random(10),
			Pages:     200 + rand.Intn(100),
			CreatedBy: creatorId,
		}
		out[i] = s.data[id]
	}
	return
}

func (s *_bookService) createBook(ctx context.Context) (out Book, err ierrors.Error) {
	var id uuid.UUID
	var param CreateBookRequest
	var modal api.RequestModal[CreateBookRequest]
	if loadError := modal.FromContext(ctx, &param); loadError != nil {
		return out, gerrors.RequestLoadError[CreateBookRequest](loadError)
	} else if id = uuid.New(); param.ID != nil {
		id = *param.ID
	}
	s.data[id] = param.book(id)
	return *s.data[id], nil
}

type ListBooksRequest = api.ListRequest

func (s *_bookService) listBooks(ctx context.Context) (out []Book, err ierrors.Error) {
	var param ListBooksRequest
	var modal api.RequestModal[api.ListRequest]
	if loadError := modal.FromContext(ctx, &param); loadError != nil {
		return out, gerrors.RequestLoadError[ListBooksRequest](loadError)
	}
	var from, to int = int(param.Offset), int(param.Limit + param.Offset)
	out = collections.ListMap(s.data.Values().Slice(from, to), collections.ListMapToNoPtrFunc[Book])
	if ginCtx, ok := ctx.(*gin.Context); ok {
		param.DefaultMetadata(ginadapter.NewRequestContext(ginCtx)).WithTotal(int64(s.data.Values().Size()))
	}
	return
}

type DetailedBookRequest struct {
	api.ListRequestWithID[uuid.UUID]
	User auth.User
}

func (s *_bookService) getBookByID(ctx context.Context) (out Book, err ierrors.Error) {
	var param DetailedBookRequest
	var modal api.RequestModal[api.ListRequestWithID[uuid.UUID]]
	if loadError := modal.FromContext(ctx, &param.ListRequestWithID); loadError != nil {
		return out, gerrors.RequestLoadError[DetailedBookRequest](loadError)
	} else if book, ok := s.data[param.ID]; !ok {
		return out, gerrors.NotFoundError[Book](param)
	} else {
		return *book, nil
	}
}