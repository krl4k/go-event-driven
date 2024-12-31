package events

import (
	"context"
	"errors"
	"fmt"
	"tickets/internal/entities"
	"tickets/internal/repository"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/google/uuid"
)

type CommandBus interface {
	Send(ctx context.Context, command any) error
}

type EventBus interface {
	Publish(ctx context.Context, event any) error
}

type VipBundleRepository interface {
	Add(ctx context.Context, vipBundle entities.VipBundle) error
	Get(ctx context.Context, vipBundleID uuid.UUID) (entities.VipBundle, error)
	GetByBookingID(ctx context.Context, bookingID uuid.UUID) (entities.VipBundle, error)

	UpdateByID(
		ctx context.Context,
		id uuid.UUID,
		updateFn func(vipBundle entities.VipBundle) (entities.VipBundle, error),
	) (entities.VipBundle, error)

	UpdateByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle entities.VipBundle) (entities.VipBundle, error),
	) (entities.VipBundle, error)
}

// type VipBundleProcessManager struct {
// 	commandBus *cqrs.CommandBus
// 	eventBus   *cqrs.EventBus
// 	repository VipBundleRepository
// }

// func NewVipBundleProcessManager(
// 	commandBus *cqrs.CommandBus,
// 	eventBus *cqrs.EventBus,
// 	repository VipBundleRepository,
// ) *VipBundleProcessManager {
// 	return &VipBundleProcessManager{
// 		commandBus: commandBus,
// 		eventBus:   eventBus,
// 		repository: repository,
// 	}
// }

// func (v VipBundleProcessManager) OnVipBundleInitialized(ctx context.Context, event *entities.VipBundleInitialized_v1) error {
// 	vb, err := v.repository.Get(ctx, event.VipBundleID)
// 	if err != nil {
// 		return err
// 	}

// 	return v.commandBus.Send(ctx, entities.BookShowTickets{
// 		BookingID:       vb.BookingID,
// 		CustomerEmail:   vb.CustomerEmail,
// 		NumberOfTickets: vb.NumberOfTickets,
// 		ShowId:          vb.ShowId,
// 	})
// }

// func (v VipBundleProcessManager) OnBookingMade(ctx context.Context, event *entities.BookingMade_v1) error {
// 	vb, err := v.repository.UpdateByBookingID(
// 		ctx,
// 		event.BookingID,
// 		func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
// 			vipBundle.BookingMadeAt = &event.Header.PublishedAt
// 			return vipBundle, nil
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	return v.commandBus.Send(ctx, entities.BookFlight{
// 		CustomerEmail:  vb.CustomerEmail,
// 		FlightID:       vb.InboundFlightID,
// 		Passengers:     vb.Passengers,
// 		ReferenceID:    vb.VipBundleID.String(),
// 		IdempotencyKey: uuid.NewString(),
// 	})
// }

// func (v VipBundleProcessManager) OnTicketBookingConfirmed(ctx context.Context, event *entities.TicketBookingConfirmed_v1) error {
// 	_, err := v.repository.UpdateByBookingID(
// 		ctx,
// 		uuid.MustParse(event.BookingID),
// 		func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
// 			eventTicketID := uuid.MustParse(event.TicketID)

// 			for _, ticketID := range vipBundle.TicketIDs {
// 				if ticketID == eventTicketID {
// 					// re-delivery (already stored)
// 					continue
// 				}
// 			}

// 			vipBundle.TicketIDs = append(vipBundle.TicketIDs, eventTicketID)

// 			return vipBundle, nil
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (v VipBundleProcessManager) OnBookingFailed(ctx context.Context, event *entities.BookingFailed_v1) error {
// 	vb, err := v.repository.GetByBookingID(ctx, event.BookingID)
// 	if err != nil {
// 		return err
// 	}

// 	return v.rollbackProcess(ctx, vb.VipBundleID)
// }

// func (v VipBundleProcessManager) OnFlightBooked(ctx context.Context, event *entities.FlightBooked_v1) error {
// 	vb, err := v.repository.UpdateByID(
// 		ctx,
// 		uuid.MustParse(event.ReferenceID),
// 		func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
// 			if vipBundle.InboundFlightID == event.FlightID {
// 				vipBundle.InboundFlightBookedAt = &event.Header.PublishedAt
// 				vipBundle.InboundFlightTicketsIDs = event.TicketIDs
// 			}
// 			if vipBundle.ReturnFlightID == event.FlightID {
// 				vipBundle.ReturnFlightBookedAt = &event.Header.PublishedAt
// 				vipBundle.ReturnFlightTicketsIDs = event.TicketIDs
// 			}

// 			return vipBundle, nil
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	switch {
// 	case vb.InboundFlightBookedAt != nil && vb.ReturnFlightBookedAt == nil:
// 		return v.commandBus.Send(ctx, entities.BookFlight{
// 			CustomerEmail:  vb.CustomerEmail,
// 			FlightID:       vb.ReturnFlightID,
// 			Passengers:     vb.Passengers,
// 			ReferenceID:    vb.VipBundleID.String(),
// 			IdempotencyKey: uuid.NewString(),
// 		})
// 	case vb.InboundFlightBookedAt != nil && vb.ReturnFlightBookedAt != nil:
// 		return v.commandBus.Send(ctx, entities.BookTaxi{
// 			CustomerEmail:      vb.CustomerEmail,
// 			CustomerName:       vb.Passengers[0],
// 			NumberOfPassengers: vb.NumberOfTickets,
// 			ReferenceID:        vb.VipBundleID.String(),
// 			IdempotencyKey:     uuid.NewString(),
// 		})
// 	default:
// 		return fmt.Errorf(
// 			"unsupported state: InboundFlightBookedAt: %v, ReturnFlightBookedAt: %v",
// 			vb.InboundFlightBookedAt,
// 			vb.ReturnFlightBookedAt,
// 		)
// 	}
// }

// func (v VipBundleProcessManager) OnFlightBookingFailed(ctx context.Context, event *entities.FlightBookingFailed_v1) error {
// 	return v.rollbackProcess(ctx, uuid.MustParse(event.ReferenceID))
// }

// func (v VipBundleProcessManager) OnTaxiBooked(ctx context.Context, event *entities.TaxiBooked_v1) error {
// 	vb, err := v.repository.UpdateByID(
// 		ctx,
// 		uuid.MustParse(event.ReferenceID),
// 		func(vb entities.VipBundle) (entities.VipBundle, error) {
// 			vb.TaxiBookedAt = &event.Header.PublishedAt
// 			vb.TaxiBookingID = &event.TaxiBookingID

// 			vb.IsFinalized = true

// 			return vb, nil
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	return v.eventBus.Publish(ctx, entities.VipBundleFinalized_v1{
// 		Header:      entities.NewEventHeader(),
// 		VipBundleID: vb.VipBundleID,
// 	})
// }

// func (v VipBundleProcessManager) OnTaxiBookingFailed(ctx context.Context, event *entities.TaxiBookingFailed_v1) error {
// 	return v.rollbackProcess(ctx, uuid.MustParse(event.ReferenceID))
// }

// func (v VipBundleProcessManager) rollbackProcess(ctx context.Context, vipBundleID uuid.UUID) error {
// 	vb, err := v.repository.Get(ctx, vipBundleID)
// 	if err != nil {
// 		return err
// 	}

// 	if vb.BookingMadeAt != nil {
// 		if err := v.rollbackTickets(ctx, vb); err != nil {
// 			return err
// 		}
// 	}
// 	if vb.InboundFlightBookedAt != nil {
// 		if err := v.commandBus.Send(ctx, entities.CancelFlightTickets{
// 			FlightTicketIDs: vb.InboundFlightTicketsIDs,
// 		}); err != nil {
// 			return err
// 		}
// 	}
// 	if vb.ReturnFlightBookedAt != nil {
// 		if err := v.commandBus.Send(ctx, entities.CancelFlightTickets{
// 			FlightTicketIDs: vb.ReturnFlightTicketsIDs,
// 		}); err != nil {
// 			return err
// 		}
// 	}

// 	_, err = v.repository.UpdateByID(
// 		ctx,
// 		vb.VipBundleID,
// 		func(vb entities.VipBundle) (entities.VipBundle, error) {
// 			vb.IsFinalized = true
// 			vb.Failed = true
// 			return vb, nil
// 		},
// 	)

// 	return err
// }

// func (v VipBundleProcessManager) rollbackTickets(ctx context.Context, vb entities.VipBundle) error {
// 	// TicketIDs is eventually consistent, we need to ensure that all tickets are stored
// 	// for alternative solutions please check "Message Ordering" module
// 	if len(vb.TicketIDs) != vb.NumberOfTickets {
// 		return fmt.Errorf(
// 			"invalid number of tickets, expected %d, has %d: not all of TicketBookingConfirmed_v1 events were processed",
// 			vb.NumberOfTickets,
// 			len(vb.TicketIDs),
// 		)
// 	}

// 	for _, ticketID := range vb.TicketIDs {
// 		if err := v.commandBus.Send(ctx, entities.RefundTicket{
// 			Header:   entities.NewEventHeader(),
// 			TicketID: ticketID.String(),
// 		}); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

type VipBundleProcessManager struct {
	commandBus CommandBus
	eventBus   EventBus
	repository VipBundleRepository
}

func NewVipBundleProcessManager(
	commandBus CommandBus,
	eventBus EventBus,
	repository VipBundleRepository,
) *VipBundleProcessManager {
	return &VipBundleProcessManager{
		commandBus: commandBus,
		eventBus:   eventBus,
		repository: repository,
	}
}

func (v VipBundleProcessManager) OnVipBundleInitialized(ctx context.Context, event *entities.VipBundleInitialized_v1) error {
	vpBundle, err := v.repository.Get(ctx, event.VipBundleID)
	if err != nil {
		return fmt.Errorf("OnVipBundleInitialized: get vip bundle: %w", err)
	}

	err = v.commandBus.Send(ctx, entities.BookShowTickets{
		BookingID:       vpBundle.BookingID,
		CustomerEmail:   vpBundle.CustomerEmail,
		NumberOfTickets: vpBundle.NumberOfTickets,
		ShowId:          vpBundle.ShowId,
	})
	if err != nil {
		return fmt.Errorf("OnVipBundleInitialized: sending book show tickets: %w", err)
	}

	return err
}

func (v VipBundleProcessManager) OnBookingMade(ctx context.Context, event *entities.BookingMade_v1) error {
	vpBundle, err := v.repository.UpdateByBookingID(ctx, event.BookingID, func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
		vipBundle.BookingMadeAt = &event.Header.PublishedAt
		return vipBundle, nil
	})
	if err != nil {
		// if errors.Is(err, repository.ErrVipBundleSkipped) {
		// 	return nil
		// }
		return fmt.Errorf("OnBookingMade: update vip bundle: %w", err)
	}

	// book inbound flight
	err = v.commandBus.Send(ctx, entities.BookFlight{
		CustomerEmail:  vpBundle.CustomerEmail,
		FlightID:       vpBundle.InboundFlightID,
		Passengers:     vpBundle.Passengers,
		ReferenceID:    vpBundle.VipBundleID.String(),
		IdempotencyKey: uuid.New().String(),
	})
	if err != nil {
		return fmt.Errorf("OnBookingMade: sending book flight: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnTicketBookingConfirmed(ctx context.Context, event *entities.TicketBookingConfirmed_v1) error {
	bookingID, err := uuid.Parse(event.BookingID)
	if err != nil {
		return fmt.Errorf("OnTicketBookingConfirmed: parse booking id: %w", err)
	}

	_, err = v.repository.UpdateByBookingID(ctx, bookingID, func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
		eventTicketID := uuid.MustParse(event.TicketID)

		for _, ticketID := range vipBundle.TicketIDs {
			if ticketID == eventTicketID {
				// re-delivery (already stored)
				continue
			}
		}

		vipBundle.TicketIDs = append(vipBundle.TicketIDs, eventTicketID)

		return vipBundle, nil
	})
	if err != nil {
		// if errors.Is(err, repository.ErrVipBundleSkipped) {
		// return nil
		// }
		return fmt.Errorf("OnTicketBookingConfirmed: update vip bundle: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnFlightBooked(ctx context.Context, event *entities.FlightBooked_v1) error {
	vb, err := v.repository.UpdateByID(
		ctx,
		uuid.MustParse(event.ReferenceID),
		func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
			if vipBundle.InboundFlightID == event.FlightID {
				vipBundle.InboundFlightBookedAt = &event.Header.PublishedAt
				vipBundle.InboundFlightTicketsIDs = event.TicketIDs
			}
			if vipBundle.ReturnFlightID == event.FlightID {
				vipBundle.ReturnFlightBookedAt = &event.Header.PublishedAt
				vipBundle.ReturnFlightTicketsIDs = event.TicketIDs
			}

			return vipBundle, nil
		},
	)
	if err != nil {
		return err
	}

	switch {
	case vb.InboundFlightBookedAt != nil && vb.ReturnFlightBookedAt == nil:
		return v.commandBus.Send(ctx, entities.BookFlight{
			CustomerEmail:  vb.CustomerEmail,
			FlightID:       vb.ReturnFlightID,
			Passengers:     vb.Passengers,
			ReferenceID:    vb.VipBundleID.String(),
			IdempotencyKey: uuid.NewString(),
		})
	case vb.InboundFlightBookedAt != nil && vb.ReturnFlightBookedAt != nil:
		return v.commandBus.Send(ctx, entities.BookTaxi{
			CustomerEmail:      vb.CustomerEmail,
			CustomerName:       vb.Passengers[0],
			NumberOfPassengers: vb.NumberOfTickets,
			ReferenceID:        vb.VipBundleID.String(),
			IdempotencyKey:     uuid.NewString(),
		})
	default:
		return fmt.Errorf(
			"unsupported state: InboundFlightBookedAt: %v, ReturnFlightBookedAt: %v",
			vb.InboundFlightBookedAt,
			vb.ReturnFlightBookedAt,
		)
	}
}

func (v VipBundleProcessManager) OnTaxiBooked(ctx context.Context, event *entities.TaxiBooked_v1) error {
	vpBundleID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnTaxiBooked: parse reference id: %w", err)
	}
	_, err = v.repository.UpdateByID(ctx, vpBundleID, func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
		vipBundle.TaxiBookedAt = &event.Header.PublishedAt
		vipBundle.TaxiBookingID = &event.TaxiBookingID
		vipBundle.IsFinalized = true
		return vipBundle, nil
	})
	if err != nil {
		// if errors.Is(err, repository.ErrVipBundleSkipped) {
		// 	return nil
		// }
		return fmt.Errorf("OnTaxiBooked: update vip bundle: %w", err)
	}

	err = v.eventBus.Publish(ctx, entities.VipBundleFinalized_v1{
		Header:      entities.NewEventHeader(),
		VipBundleID: vpBundleID,
	})
	if err != nil {
		return fmt.Errorf("OnTaxiBooked: sending vip bundle finalized: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnBookingFailed(ctx context.Context, event *entities.BookingFailed_v1) error {
	vpBundle, err := v.repository.GetByBookingID(ctx, event.BookingID)
	if err != nil {
		if errors.Is(err, repository.ErrVipBundleNotFound) {
			return nil
		}
		return fmt.Errorf("OnBookingFailed: get vip bundle: %w", err)
	}

	err = v.rollback(ctx, vpBundle)
	if err != nil {
		return fmt.Errorf("OnBookingFailed: rollback: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnFlightBookingFailed(ctx context.Context, event *entities.FlightBookingFailed_v1) error {
	vpBundleID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnFlightBookingFailed: parse reference id: %w", err)
	}
	vpBundle, err := v.repository.Get(ctx, vpBundleID)
	if err != nil {
		if errors.Is(err, repository.ErrVipBundleNotFound) {
			return nil
		}
		return fmt.Errorf("OnFlightBookingFailed: get vip bundle: %w", err)
	}

	err = v.rollback(ctx, vpBundle)
	if err != nil {
		return fmt.Errorf("OnFlightBookingFailed: rollback: %w", err)
	}
	return nil
}

func (v VipBundleProcessManager) OnTaxiBookingFailed(ctx context.Context, event *entities.TaxiBookingFailed_v1) error {
	vpBundleID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnTaxiBookingFailed: parse reference id: %w", err)
	}
	vpBundle, err := v.repository.Get(ctx, vpBundleID)
	if err != nil {
		if errors.Is(err, repository.ErrVipBundleNotFound) {
			return nil
		}
		return fmt.Errorf("OnTaxiBookingFailed: get vip bundle: %w", err)
	}

	err = v.rollback(ctx, vpBundle)
	if err != nil {
		return fmt.Errorf("OnTaxiBookingFailed: rollback: %w", err)
	}
	return nil
}

func (v VipBundleProcessManager) rollback(ctx context.Context, vpBundle entities.VipBundle) error {
	if vpBundle.BookingMadeAt != nil &&
		len(vpBundle.TicketIDs) != vpBundle.NumberOfTickets {
		return fmt.Errorf("rollback: number of tickets and ticket ids mismatch, number of tickets: %d, ticketsID count: %d", vpBundle.NumberOfTickets, len(vpBundle.TicketIDs))
	}

	log.FromContext(ctx).Info("Rollback: all tickets received")

	for _, ticketID := range vpBundle.TicketIDs {
		err := v.commandBus.Send(ctx, entities.RefundTicket{
			Header:   entities.NewEventHeader(),
			TicketID: ticketID.String(),
		})
		if err != nil {
			return fmt.Errorf("rollback: sending refund ticket: %w", err)
		}
	}

	if vpBundle.InboundFlightTicketsIDs != nil {
		// rollback inbound flight
		err := v.commandBus.Send(ctx, entities.CancelFlightTickets{
			FlightTicketIDs: vpBundle.InboundFlightTicketsIDs,
		})
		if err != nil {
			return fmt.Errorf("rollback: sending cancel inbound flight tickets: %w", err)
		}
	}

	if vpBundle.ReturnFlightTicketsIDs != nil {
		// rollback return flight
		err := v.commandBus.Send(ctx, entities.CancelFlightTickets{
			FlightTicketIDs: vpBundle.ReturnFlightTicketsIDs,
		})
		if err != nil {
			return fmt.Errorf("rollback: sending cancel return flight tickets: %w", err)
		}
	}

	_, err := v.repository.UpdateByBookingID(ctx, vpBundle.BookingID, func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
		vipBundle.Failed = true
		vipBundle.IsFinalized = true
		return vipBundle, nil
	})
	if err != nil {
		// if errors.Is(err, repository.ErrVipBundleSkipped) {
		// 	return nil
		// }
		return fmt.Errorf("rollback: update vip bundle: %w", err)
	}
	return nil
}
