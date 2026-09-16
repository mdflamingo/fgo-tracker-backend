package postgres

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
)

func (d *DBStorage) GetUserList() ([]model.UserDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT id, username, email
			FROM "user"
			ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}
	defer rows.Close()

	usersList := make([]model.UserDB, 0)

	for rows.Next() {
		var user model.UserDB
		if err := rows.Scan(&user.Id, &user.Username, &user.Email); err != nil {
			return nil, fmt.Errorf("data scan error: %w", err)
		}
		usersList = append(usersList, user)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows processing error: %w", err)
	}

	return usersList, nil
}
