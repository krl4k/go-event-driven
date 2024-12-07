package commands

import "github.com/ThreeDotsLabs/watermill/components/cqrs"

type Handler struct {
	eb *cqrs.CommandBus
}

func NewHandler(eb *cqrs.CommandBus) *Handler {
	return &Handler{
		eb: eb,
	}
}
