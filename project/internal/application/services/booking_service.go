package services

import (
	"context"
	"github.com/google/uuid"
	domain "tickets/internal/domain/bookings"
)

type BookingRepo interface {
	CreateBooking(ctx context.Context, booking domain.Booking) (uuid.UUID, error)
}

type BookingService struct {
	bookingRepo BookingRepo
}

func NewBookingService(bookingRepo BookingRepo) *BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
	}
}

func (s *BookingService) BookTickets(ctx context.Context, booking domain.Booking) (uuid.UUID, error) {
	return s.bookingRepo.CreateBooking(ctx, booking)
}
