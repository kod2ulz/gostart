package api_test

import (
	"context"
	"errors"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/utils"
)

type Book struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Author    string     `json:"author"`
	Pages     int        `json:"pages"`
	CreatedBy *uuid.UUID `json:"createdBy,omitempty"`
}

type CreateBookRequest struct {
	ID     *uuid.UUID `json:"id,omitempty"`
	User   api.User   `json:"-"`
	Name   string     `json:"name"   validate:"required"`
	Author string     `json:"author" validate:"required"`
	Pages  int        `json:"pages"  validate:"required,gt=200"`
	api.RequestModal[CreateBookRequest]
}

func (r CreateBookRequest) RequestLoad(ctx context.Context) (param api.RequestParam, err error) {
	var out CreateBookRequest
	if err = out.LoadFromJsonBody(ctx, &out); err != nil {
		return param, err
	}
	out.User, _ = api.GetUser(ctx) // ignoring error because some tests won't need r.User
	ctx.(*gin.Context).Set(out.ContextKey(), out)
	return out, nil
}

func (r *CreateBookRequest) book(id uuid.UUID) (out *Book) {
	out = &Book{ID: id, Name: r.Name, Author: r.Author, Pages: r.Pages}
	if r.User != nil {
		out.CreatedBy = utils.PointerTo(r.User.ID())
	}
	return
}

func bookService() *_bookService {
	return &_bookService{make(collections.Map[uuid.UUID, *Book])}
}

type _bookService struct {
	data collections.Map[uuid.UUID, *Book]
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

func (s *_bookService) seed(size int, user api.User) (out []*Book, err error) {
	var creatorId *uuid.UUID
	if size < 1 {
		return out, errors.New("seed size is required and cannot be 0")
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

func (s *_bookService) createBook(ctx context.Context) (out Book, err api.Error) {
	var id uuid.UUID
	var param CreateBookRequest
	if loadError := param.FromContext(ctx, &param); loadError != nil {
		return out, api.RequestLoadError[CreateBookRequest](loadError)
	} else if id = uuid.New(); param.ID != nil {
		id = *param.ID
	}
	s.data[id] = param.book(id)
	return *s.data[id], nil
}

type ListBooksRequest = api.ListRequest

func (s *_bookService) listBooks(ctx context.Context) (out []Book, err api.Error) {
	var param ListBooksRequest
	if loadError := param.FromContext(ctx, &param); loadError != nil {
		return out, api.RequestLoadError[ListBooksRequest](loadError)
	}
	var from, to int = int(param.Offset), int(param.Limit + param.Offset)
	out = collections.ListMap(s.data.Values().Slice(from, to), collections.ListMapToNoPtrFunc[Book])
	param.DefaultMetadata(ctx).WithTotal(int64(s.data.Values().Size()))
	return
}

type DetailedBookRequest = api.ListRequestWithID[uuid.UUID]

func (s *_bookService) getBookByID(ctx context.Context) (out Book, err api.Error) {
	var param DetailedBookRequest
	if loadError := param.FromContext(ctx, &param); loadError != nil {
		return out, api.RequestLoadError[DetailedBookRequest](loadError)
	} else if book, ok := s.data[param.ID]; !ok {
		return out, api.NotFoundError[Book](param)
	} else if book.CreatedBy != nil && *book.CreatedBy != param.User.ID() {
		return out, api.ServiceErrorUnauthorised(errors.New("you cannot access this book"))
	} else {
		return *book, nil
	}
}
