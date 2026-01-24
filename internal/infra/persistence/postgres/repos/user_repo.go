package repos

import (
	"context"
	"database/sql"
	"distributed-crawler/internal/domain/auth/models"
	userrepo "distributed-crawler/internal/domain/auth/repos/user"
	"distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const (
	usersTableName      = "users"
	userIDColumn        = "id"
	userEmailColumn     = "email"
	userPasswordColumn  = "password_hash"
	userRoleColumn      = "role"
	userCreatedAtColumn = "created_at"
	userUpdatedAtColumn = "updated_at"
)

type userRepository struct {
	client persistence.Client
}

func NewUserRepository(client persistence.Client) userrepo.UserRepository {
	return &userRepository{client: client}
}

func (r *userRepository) Create(ctx context.Context, entity *models.User) (valueobjects.UserID, error) {
	dbEntity := converters.UserModelToSnapshot(entity)

	builder := sq.Insert(usersTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(userIDColumn, userEmailColumn, userPasswordColumn, userRoleColumn, userCreatedAtColumn, userUpdatedAtColumn).
		Values(dbEntity.ID, dbEntity.Email, dbEntity.PasswordHash, dbEntity.Role, dbEntity.CreatedAt, dbEntity.UpdatedAt).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.UserID{}, err
	}

	q := persistence.Query{
		Name:     "user_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = r.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.UserID{}, err
	}

	return valueobjects.NewUserID(id)
}

func (r *userRepository) GetByID(ctx context.Context, id valueobjects.UserID) (*models.User, error) {
	builder := sq.Select(userIDColumn, userEmailColumn, userPasswordColumn, userRoleColumn, userCreatedAtColumn, userUpdatedAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(usersTableName).
		Where(sq.Eq{userIDColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "user_repository.GetByID",
		QueryRaw: query,
	}

	var userSnapshot snapshots.UserSnapshot
	err = r.client.DB().ScanOneContext(ctx, &userSnapshot, q, args...)
	if err != nil {
		// Check for no rows found (user doesn't exist)
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return converters.UserSnapshotToModel(userSnapshot)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	builder := sq.Select(userIDColumn, userEmailColumn, userPasswordColumn, userRoleColumn, userCreatedAtColumn, userUpdatedAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(usersTableName).
		Where(sq.Eq{userEmailColumn: email}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "user_repository.GetByEmail",
		QueryRaw: query,
	}

	var userSnapshot snapshots.UserSnapshot
	err = r.client.DB().ScanOneContext(ctx, &userSnapshot, q, args...)
	if err != nil {
		// Check for no rows found (user doesn't exist)
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return converters.UserSnapshotToModel(userSnapshot)
}

func (r *userRepository) Update(ctx context.Context, entity *models.User) error {
	dbEntity := converters.UserModelToSnapshot(entity)

	builder := sq.Update(usersTableName).
		PlaceholderFormat(sq.Dollar).
		Set(userEmailColumn, dbEntity.Email).
		Set(userPasswordColumn, dbEntity.PasswordHash).
		Set(userRoleColumn, dbEntity.Role).
		Set(userUpdatedAtColumn, dbEntity.UpdatedAt).
		Where(sq.Eq{userIDColumn: dbEntity.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "user_repository.Update",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *userRepository) List(ctx context.Context) ([]*models.User, error) {
	builder := sq.Select(userIDColumn, userEmailColumn, userPasswordColumn, userRoleColumn, userCreatedAtColumn, userUpdatedAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(usersTableName).
		OrderBy(userCreatedAtColumn + " DESC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "user_repository.List",
		QueryRaw: query,
	}

	var snapshots []snapshots.UserSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snapshots, q, args...)
	if err != nil {
		return nil, err
	}

	users := make([]*models.User, 0, len(snapshots))
	for _, snapshot := range snapshots {
		user, err := converters.UserSnapshotToModel(snapshot)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *userRepository) UpdateRole(ctx context.Context, id valueobjects.UserID, role models.Role) error {
	builder := sq.Update(usersTableName).
		PlaceholderFormat(sq.Dollar).
		Set(userRoleColumn, string(role)).
		Set(userUpdatedAtColumn, time.Now().UTC()).
		Where(sq.Eq{userIDColumn: id.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "user_repository.UpdateRole",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}
