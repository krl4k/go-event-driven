package events

import (
	"context"
	"errors"
	"fmt"
	"tickets/internal/entities"
	"tickets/internal/repository"

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
		if errors.Is(err, repository.ErrVipBundleSkipped) {
			return nil
		}
		return fmt.Errorf("OnBookingMade: update vip bundle: %w", err)
	}

	// book inbound flight
	err = v.commandBus.Send(ctx, entities.BookFlight{
		CustomerEmail:  vpBundle.CustomerEmail,
		FlightID:       vpBundle.InboundFlightID,
		Passengers:     vpBundle.Passengers,
		ReferenceID:    vpBundle.VipBundleID.String(),
		IdempotencyKey: event.Header.IdempotencyKey,
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
		vipBundle.TicketIDs = append(vipBundle.TicketIDs, uuid.MustParse(event.TicketID))
		return vipBundle, nil
	})
	if err != nil {
		if errors.Is(err, repository.ErrVipBundleSkipped) {
			return nil
		}
		return fmt.Errorf("OnTicketBookingConfirmed: update vip bundle: %w", err)
	}

	return nil
}

func (v VipBundleProcessManager) OnFlightBooked(ctx context.Context, event *entities.FlightBooked_v1) error {
	vpBundlerID, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return fmt.Errorf("OnFlightBooked: parse reference id: %w", err)
	}
	vpBundle, err := v.repository.Get(ctx, vpBundlerID)
	if err != nil {
		return fmt.Errorf("OnFlightBooked: get vip bundle: %w", err)
	}

	switch event.FlightID {
	case vpBundle.InboundFlightID:
		vpBundle.InboundFlightBookedAt = &event.Header.PublishedAt
		vpBundle.InboundFlightTicketsIDs = event.TicketIDs

		_, err = v.repository.UpdateByID(ctx, vpBundlerID, func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
			return vpBundle, nil
		})
		if err != nil {
			if errors.Is(err, repository.ErrVipBundleSkipped) {
				return nil
			}
			return fmt.Errorf("failed to update vip bundle: %w", err)
		}

		// book return flight
		err = v.commandBus.Send(ctx, entities.BookFlight{
			CustomerEmail:  vpBundle.CustomerEmail,
			FlightID:       vpBundle.ReturnFlightID,
			Passengers:     vpBundle.Passengers,
			ReferenceID:    vpBundle.VipBundleID.String(),
			IdempotencyKey: event.Header.IdempotencyKey,
		})
		if err != nil {
			return fmt.Errorf("OnBookingMade: sending book flight: %w", err)
		}
		return nil
	case vpBundle.ReturnFlightID:
		vpBundle.ReturnFlightBookedAt = &event.Header.PublishedAt
		vpBundle.ReturnFlightTicketsIDs = event.TicketIDs
		_, err = v.repository.UpdateByID(ctx, vpBundlerID, func(vipBundle entities.VipBundle) (entities.VipBundle, error) {
			return vpBundle, nil
		})
		if err != nil {
			if errors.Is(err, repository.ErrVipBundleSkipped) {
				return nil
			}
			return fmt.Errorf("failed to update vip bundle: %w", err)
		}
	default:
		return fmt.Errorf("OnFlightBooked: unknown flight id: %s", event.FlightID.String())
	}

	if vpBundle.InboundFlightBookedAt != nil && vpBundle.ReturnFlightBookedAt != nil {
		err = v.commandBus.Send(ctx, entities.BookTaxi{
			CustomerEmail:      vpBundle.CustomerEmail,
			CustomerName:       vpBundle.Passengers[0],
			NumberOfPassengers: vpBundle.NumberOfTickets,
			ReferenceID:        vpBundle.VipBundleID.String(),
			IdempotencyKey:     event.Header.IdempotencyKey,
		})
		if err != nil {
			return fmt.Errorf("OnFlightBooked: sending book taxi: %w", err)
		}
	}

	return nil
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
		if errors.Is(err, repository.ErrVipBundleSkipped) {
			return nil
		}
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
		(vpBundle.TicketIDs == nil || len(vpBundle.TicketIDs) != vpBundle.NumberOfTickets) {
		return fmt.Errorf("rollback: number of tickets and ticket ids mismatch")
	}

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
		if errors.Is(err, repository.ErrVipBundleSkipped) {
			return nil
		}
		return fmt.Errorf("rollback: update vip bundle: %w", err)
	}

	return nil
}
