package repos

import (
	"context"
	"database/sql"
	"distributed-crawler/internal/domain/auth/models"
	refreshtokenrepo "distributed-crawler/internal/domain/auth/repos/refresh_token"
	"distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const (
	refreshTokensTableName       = "refresh_tokens"
	refreshTokenIDColumn         = "id"
	refreshTokenUserIDColumn     = "user_id"
	refreshTokenHashColumn       = "token_hash"
	refreshTokenExpiresAtColumn  = "expires_at"
	refreshTokenRevokedAtColumn  = "revoked_at"
	refreshTokenCreatedAtColumn  = "created_at"
	refreshTokenReplacedByColumn = "replaced_by_token_id"
)

type refreshTokenRepository struct {
	client persistence.Client
}

func NewRefreshTokenRepository(client persistence.Client) refreshtokenrepo.RefreshTokenRepository {
	return &refreshTokenRepository{client: client}
}

func (r *refreshTokenRepository) Create(ctx context.Context, entity *models.RefreshToken) (valueobjects.RefreshTokenID, error) {
	dbEntity := converters.RefreshTokenModelToSnapshot(entity)

	builder := sq.Insert(refreshTokensTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(
			refreshTokenIDColumn,
			refreshTokenUserIDColumn,
			refreshTokenHashColumn,
			refreshTokenExpiresAtColumn,
			refreshTokenRevokedAtColumn,
			refreshTokenCreatedAtColumn,
			refreshTokenReplacedByColumn,
		).
		Values(
			dbEntity.ID,
			dbEntity.UserID,
			dbEntity.TokenHash,
			dbEntity.ExpiresAt,
			dbEntity.RevokedAt,
			dbEntity.CreatedAt,
			dbEntity.ReplacedByTokenID,
		).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.RefreshTokenID{}, err
	}

	q := persistence.Query{
		Name:     "refresh_token_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = r.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.RefreshTokenID{}, err
	}

	return valueobjects.NewRefreshTokenID(id)
}

func (r *refreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	builder := sq.Select(
		refreshTokenIDColumn,
		refreshTokenUserIDColumn,
		refreshTokenHashColumn,
		refreshTokenExpiresAtColumn,
		refreshTokenRevokedAtColumn,
		refreshTokenCreatedAtColumn,
		refreshTokenReplacedByColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(refreshTokensTableName).
		Where(sq.Eq{refreshTokenHashColumn: tokenHash}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "refresh_token_repository.GetByTokenHash",
		QueryRaw: query,
	}

	var tokenSnapshot snapshots.RefreshTokenSnapshot
	err = r.client.DB().ScanOneContext(ctx, &tokenSnapshot, q, args...)
	if err != nil {
		// Check for no rows found (token doesn't exist)
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return converters.RefreshTokenSnapshotToModel(tokenSnapshot)
}

func (r *refreshTokenRepository) GetByID(ctx context.Context, id valueobjects.RefreshTokenID) (*models.RefreshToken, error) {
	builder := sq.Select(
		refreshTokenIDColumn,
		refreshTokenUserIDColumn,
		refreshTokenHashColumn,
		refreshTokenExpiresAtColumn,
		refreshTokenRevokedAtColumn,
		refreshTokenCreatedAtColumn,
		refreshTokenReplacedByColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(refreshTokensTableName).
		Where(sq.Eq{refreshTokenIDColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "refresh_token_repository.GetByID",
		QueryRaw: query,
	}

	var tokenSnapshot snapshots.RefreshTokenSnapshot
	err = r.client.DB().ScanOneContext(ctx, &tokenSnapshot, q, args...)
	if err != nil {
		// Check for no rows found (token doesn't exist)
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return converters.RefreshTokenSnapshotToModel(tokenSnapshot)
}

func (r *refreshTokenRepository) Update(ctx context.Context, entity *models.RefreshToken) error {
	dbEntity := converters.RefreshTokenModelToSnapshot(entity)

	builder := sq.Update(refreshTokensTableName).
		PlaceholderFormat(sq.Dollar).
		Set(refreshTokenRevokedAtColumn, dbEntity.RevokedAt).
		Set(refreshTokenReplacedByColumn, dbEntity.ReplacedByTokenID).
		Where(sq.Eq{refreshTokenIDColumn: dbEntity.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "refresh_token_repository.Update",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *refreshTokenRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	builder := sq.Update(refreshTokensTableName).
		PlaceholderFormat(sq.Dollar).
		Set(refreshTokenRevokedAtColumn, time.Now()).
		Where(sq.Eq{refreshTokenHashColumn: tokenHash})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "refresh_token_repository.RevokeByTokenHash",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID valueobjects.UserID) error {
	builder := sq.Update(refreshTokensTableName).
		PlaceholderFormat(sq.Dollar).
		Set(refreshTokenRevokedAtColumn, time.Now()).
		Where(sq.Eq{refreshTokenUserIDColumn: userID.String()}).
		Where(sq.Eq{refreshTokenRevokedAtColumn: nil})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "refresh_token_repository.RevokeAllByUserID",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}
