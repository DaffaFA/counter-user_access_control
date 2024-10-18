package user

import (
	"context"
	"time"

	"github.com/DaffaFA/counter-user_access_control/pkg/entities"
	"github.com/DaffaFA/counter-user_access_control/utils"
)

type Service interface {
	GetUser(context.Context, int64) (entities.User, error)
	SignIn(context.Context, entities.User) (string, entities.User, time.Duration, error)
	Register(context.Context, entities.User) error
	SignOut(context.Context, string) error
	FetchUserSession(context.Context, string) (entities.User, error)
}

type service struct {
	repository Repository
}

// NewRepo is the single instance repo that is being created.
func NewService(repo Repository) Service {
	return &service{
		repository: repo,
	}
}

func (s *service) GetUser(ctx context.Context, userId int64) (entities.User, error) {
	ctx, span := utils.Tracer.Start(ctx, "user_access_control.service.GetUser")
	defer span.End()

	userPagination, err := s.repository.FetchUser(ctx, &entities.FetchFilter{
		ID: userId,
	})
	if err != nil {
		span.RecordError(err)
		return entities.User{}, err
	}

	if userPagination.Total == 0 {
		return entities.User{}, err
	}

	return userPagination.Users[0], nil
}

func (s *service) SignIn(ctx context.Context, user entities.User) (string, entities.User, time.Duration, error) {
	ctx, span := utils.Tracer.Start(ctx, "user_access_control.service.SignIn")
	defer span.End()

	session, user, sessionExpired, err := s.repository.SignIn(ctx, &user)
	if err != nil {
		span.RecordError(err)
		return "", entities.User{}, 0, err
	}

	return session, user, sessionExpired, nil
}

func (s *service) Register(ctx context.Context, user entities.User) error {
	ctx, span := utils.Tracer.Start(ctx, "user_access_control.service.Register")
	defer span.End()

	if err := s.repository.Register(ctx, &user); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (s *service) SignOut(ctx context.Context, session string) error {
	ctx, span := utils.Tracer.Start(ctx, "user_access_control.service.Logout")
	defer span.End()

	if err := s.repository.SignOut(ctx, session); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (s *service) FetchUserSession(ctx context.Context, session string) (entities.User, error) {
	ctx, span := utils.Tracer.Start(ctx, "user_access_control.service.FetchUserSession")
	defer span.End()

	user, err := s.repository.FetchUserSession(ctx, session)
	if err != nil {
		span.RecordError(err)
		return entities.User{}, err
	}

	return user, nil
}
