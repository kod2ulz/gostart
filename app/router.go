package app

import "context"

type Context interface {
	context.Context
	Get(key string) (value any, exists bool)
}

type HandlerFunc func(Context)

type RouterEngine[ctx Context] interface {

}