package main

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

type FollowRequestSent struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type EventsCounter interface {
	CountEvent() error
}

type FollowRequestSentHandler struct {
	Counter EventsCounter
}

func (f *FollowRequestSentHandler) HandlerName() string {
	return "follow_request_handler"
}

func (f *FollowRequestSentHandler) NewEvent() any {
	return &FollowRequestSent{}
}

func (f *FollowRequestSentHandler) Handle(ctx context.Context, event any) error {

	return f.Counter.CountEvent()
}

func NewFollowRequestSentHandler(counter EventsCounter) cqrs.EventHandler {
	return &FollowRequestSentHandler{
		Counter: counter,
	}
}
