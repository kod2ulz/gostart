package api_test

import (
	"context"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/utils"
)

type Book struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Author string    `json:"author"`
	Pages  int       `json:"pages"`
}

type CreateBookRequest struct {
	ID     *uuid.UUID `json:"id"`
	Name   string     `json:"name"   validate:"required"`
	Author string     `json:"author" validate:"required"`
	Pages  int        `json:"pages"  validate:"required,gt=200"`
	api.RequestModal[CreateBookRequest]
}

func bookService() *_bookService {
	return &_bookService{make(collections.Map[uuid.UUID, Book])}
}

type _bookService struct {
	data collections.Map[uuid.UUID, Book]
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
		POST("", api.HandlerWithResponse[CreateBookRequest](s.createBook))
}

func (s *_bookService) seedRandomBooks(size int) {
	if size < 1 {
		return
	}
	for i := 0; i < size; i ++ {
		id := uuid.New()
		s.data[id] = Book{
			ID:     id,
			Name:   utils.String.Random(20),
			Author: utils.String.Random(10),
			Pages:  200 + rand.Intn(100),
		}
	}
}

func (s *_bookService) createBook(ctx context.Context) (out Book, err api.Error) {
	var id uuid.UUID
	var param CreateBookRequest
	if loadError := param.LoadFromContext(ctx, &param); err != nil {
		err = api.RequestLoadError[CreateBookRequest](loadError)
	} else if id = uuid.New(); param.ID != nil {
		id = *param.ID
	}
	s.data[id] = Book{ID: id, Name: param.Name, Author: param.Author, Pages: param.Pages}
	return s.data[id], err
}
