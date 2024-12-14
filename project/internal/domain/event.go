package domain

type Event interface {
	IsInternal() bool
}
