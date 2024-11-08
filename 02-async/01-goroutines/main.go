package main

import "time"

type User struct {
	Email string
}

type UserRepository interface {
	CreateUserAccount(u User) error
}

type NotificationsClient interface {
	SendNotification(u User) error
}

type NewsletterClient interface {
	AddToNewsletter(u User) error
}

type Handler struct {
	repository          UserRepository
	newsletterClient    NewsletterClient
	notificationsClient NotificationsClient
}

func NewHandler(
	repository UserRepository,
	newsletterClient NewsletterClient,
	notificationsClient NotificationsClient,
) Handler {
	return Handler{
		repository:          repository,
		newsletterClient:    newsletterClient,
		notificationsClient: notificationsClient,
	}
}

func (h Handler) SignUp(u User) error {
	if err := h.repository.CreateUserAccount(u); err != nil {
		return err
	}

	// we dont want to block the user here
	asyncRun(func() error {
		return h.newsletterClient.AddToNewsletter(u)
	})
	asyncRun(func() error {
		return h.notificationsClient.SendNotification(u)
	})

	return nil
}

func asyncRun(f func() error) {
	go func() {
		for {
			if err := f(); err == nil {
				return
			}
			time.Sleep(1500 * time.Millisecond)
		}
	}()
}
