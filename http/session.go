package http

type Session interface {
	Authorization() string
}
