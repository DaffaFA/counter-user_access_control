package user

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/DaffaFA/counter-user_access_control/pkg/entities"
	"github.com/DaffaFA/counter-user_access_control/utils"
	"github.com/Masterminds/squirrel"
	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/redis/go-redis/v9"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type Repository interface {
	FetchUser(context.Context, *entities.FetchFilter) (entities.UserPagination, error)
	SignIn(context.Context, *entities.User) (string, entities.User, time.Duration, error)
	Register(context.Context, *entities.User) error
	SignOut(context.Context, string) error
	FetchUserSession(context.Context, string) (entities.User, error)
}

type repository struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

// NewRepo is the single instance repo that is being created.
func NewRepo(db *pgxpool.Pool, redis *redis.Client) Repository {
	return &repository{
		DB:    db,
		Redis: redis,
	}
}

func (r *repository) FetchUser(ctx context.Context, filter *entities.FetchFilter) (res entities.UserPagination, err error) {
	ctx, span := utils.Tracer.Start(ctx, "user.repository.FetchUser")
	defer span.End()

	users := []entities.User{}

	query := psql.Select("id", "name").From(`"order".users c`)

	entities.SetDefaultFilter(filter)

	query = query.Limit(filter.Limit).Offset(filter.Cursor - 1)

	if filter.Query != "" {
		query = query.Where("o.style_name ILIKE ?", "%"+filter.Query+"%")
	}

	if len(filter.Sort) > 0 {
		for _, sort := range filter.Sort {
			if sort[0] == '-' {
				query = query.OrderBy(sort[1:] + " DESC")
			} else {
				query = query.OrderBy(sort + " ASC")
			}
		}
	}

	sqln, in, err := query.ToSql()
	if err != nil {
		span.RecordError(err)
		return res, err
	}

	rows, err := r.DB.Query(ctx, sqln, in...)
	if err != nil {
		span.RecordError(err)
		return res, err
	}

	for rows.Next() {
		var user entities.User

		if err := rows.Scan(&user.ID); err != nil {
			span.RecordError(err)
			return res, err
		}

		users = append(users, user)
	}

	res.Users = users

	return res, nil
}

func (r *repository) SignIn(ctx context.Context, user *entities.User) (session string, res entities.User, sessionExpired time.Duration, err error) {
	ctx, span := utils.Tracer.Start(ctx, "user.repository.SignIn")
	defer span.End()

	query := psql.Select("password").From(`user_access_control.users`).Where("username = ?", user.Username)

	sqln, in, err := query.ToSql()
	if err != nil {
		span.RecordError(err)
		return session, res, sessionExpired, err
	}

	var password string

	err = r.DB.QueryRow(ctx, sqln, in...).Scan(&password)
	if err == pgx.ErrNoRows {
		return session, res, sessionExpired, errors.New("user not found")
	}

	if err != nil {
		span.RecordError(err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(user.Password)); err != nil {
		return session, res, sessionExpired, errors.New("password not match")
	}

	sqln, in, err = newUserQuery(squirrel.Eq{
		"u.username": user.Username,
	}, 1, 0).ToSql()
	if err != nil {
		span.RecordError(err)
		return session, res, sessionExpired, err
	}

	var users []entities.User
	var t int
	if err := r.DB.QueryRow(ctx, sqln, in...).Scan(&t, &users); err != nil {
		span.RecordError(err)
		return session, res, sessionExpired, err
	}

	session = gonanoid.Must(16)
	sessionExpired = time.Second * 60 * 60 * 24 * 7

	userObject, err := sonic.MarshalString(users[0])
	if err != nil {
		span.RecordError(err)
		return session, res, sessionExpired, err
	}

	if err := r.Redis.Set(ctx, session, userObject, sessionExpired).Err(); err != nil {
		span.RecordError(err)
		return session, res, sessionExpired, err
	}

	return session, users[0], sessionExpired, nil
}

func (r *repository) Register(ctx context.Context, user *entities.User) error {
	ctx, span := utils.Tracer.Start(ctx, "user.repository.Register")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		span.RecordError(err)
		return err
	}

	query := psql.Insert("user_access_control.users").Columns(
		"department_id",
		"full_name",
		"username",
		"password",
		"expired_at",
		"activated_at",
	).Values(user.DepartmentID, user.FullName, user.Username, hash, user.ExpiredAt, squirrel.Expr("COALESCE(?, NOW())", user.ActivatedAt))

	sqln, in, err := query.ToSql()
	if err != nil {
		span.RecordError(err)
		return err
	}

	if _, err = r.DB.Exec(ctx, sqln, in...); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *repository) SignOut(ctx context.Context, session string) error {
	ctx, span := utils.Tracer.Start(ctx, "user.repository.SignOut")
	defer span.End()

	if err := r.Redis.Del(ctx, session).Err(); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *repository) FetchUserSession(ctx context.Context, session string) (res entities.User, err error) {
	ctx, span := utils.Tracer.Start(ctx, "user.repository.FetchUserSession")
	defer span.End()

	userString, err := r.Redis.Get(ctx, session).Result()
	if err != nil {
		span.RecordError(err)
		return res, err
	}

	if err := sonic.UnmarshalString(userString, &res); err != nil {
		span.RecordError(err)
		return res, err
	}

	return res, nil
}

func newUserQuery(where map[string]interface{}, limit uint64, offset uint64) squirrel.SelectBuilder {
	// Subquery 1: raw_data
	rawData := psql.
		Select(
			"u.id",
			"u.department_id",
			"u.full_name",
			"u.username",
			"u.expired_at",
			"u.activated_at",
			"u.created_at",
			"u.updated_at",
			"row_to_json(d) as department",
		).
		From("user_access_control.users u").
		LeftJoin("user_access_control.departments d ON u.department_id = d.id").Where(where)

	// Subquery 2: permissions
	permissionData := psql.
		Select(
			"rd.id",
			"jsonb_object_agg(p.alias, jsonb_build_object('read', udp.read, 'write', udp.write)) FILTER (WHERE p.alias IS NOT NULL) as data",
		).
		From("raw_data rd").
		LeftJoin("user_access_control.user_department_permissions udp ON rd.department_id = udp.department_id OR udp.user_id = rd.id").
		LeftJoin("user_access_control.permissions p ON udp.permission_id = p.id").
		GroupBy("rd.id")

	// Subquery 3: data
	data := psql.
		Select(
			"rd.id",
			"rd.full_name",
			"rd.username",
			"rd.expired_at",
			"rd.activated_at",
			"rd.created_at",
			"rd.updated_at",
			"rd.department",
			"p.data as permissions",
		).
		From("raw_data rd").
		LeftJoin("permission_data p ON rd.id = p.id")

	pagination := psql.Select("*").From("data").Limit(limit).Offset(offset)

	// Final query
	query := psql.
		Select(
			"(SELECT count(id) FROM data) as total",
			"(SELECT json_agg(pagination) FROM pagination) as data",
		).
		PrefixExpr(rawData.Prefix("WITH raw_data AS (").Suffix("),")).
		PrefixExpr(permissionData.Prefix("permission_data AS (").Suffix("),")).
		PrefixExpr(data.Prefix("data AS (").Suffix("),")).
		PrefixExpr(pagination.Prefix("pagination AS (").Suffix(")"))

	return query
}
