package mq

// InitFunc is a function that is used to initialise something.
// this is meant to be used by service initialisers
type InitFunc[T any] func(*T)