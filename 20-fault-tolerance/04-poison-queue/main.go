package main

import (
	"context"
	"errors"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type orderStorage interface {
	AddTrackingLink(ctx context.Context, orderID string, trackingLink string) error
}

type OrderDispatched struct {
	OrderID      string `json:"order_id"`
	TrackingLink string `json:"tracking_link"`
}

var ErrTrackingLinkEmpty = errors.New("tracking link is empty")

func ProcessMessages(
	ctx context.Context,
	sub message.Subscriber,
	pub message.Publisher,
	storage orderStorage,
) error {
	logger := watermill.NewStdLogger(false, false)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return err
	}

	pq, err := middleware.PoisonQueueWithFilter(pub, "PoisonQueue",
		func(err error) bool {
			if errors.Is(err, ErrTrackingLinkEmpty) {
				return true
			}

			return false
		})

	router.AddMiddleware(pq)

	ep, err := cqrs.NewEventProcessorWithConfig(
		router,
		cqrs.EventProcessorConfig{
			GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
				return params.EventName, nil
			},
			SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				return sub, nil
			},
			Marshaler: cqrs.JSONMarshaler{},
			Logger:    logger,
		},
	)
	if err != nil {
		return err
	}

	err = ep.AddHandlers(
		cqrs.NewEventHandler("OnOrderDispatched", func(ctx context.Context, event *OrderDispatched) error {
			if event.TrackingLink == "" {
				return ErrTrackingLinkEmpty
			}
			return storage.AddTrackingLink(ctx, event.OrderID, event.TrackingLink)
		}),
	)
	if err != nil {
		return err
	}

	go func() {
		err := router.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	<-router.Running()

	return nil
}
