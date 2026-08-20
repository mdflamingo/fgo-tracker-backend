package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DBStorage{pool: pool}, nil
}

func (d *DBStorage) Close() error {
	d.pool.Close()
	return nil
}

func runMigrations(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (d *DBStorage) Create(task model.TaskCreate, userTaskList []model.TaskUserCreate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO task (id, name, description, status, priority, project_id, deadline)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		task.Id, task.Name, task.Description, task.Status, task.Priority, task.ProjectId, task.Deadline)

	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	if len(userTaskList) > 0 {
		ids := make([]uuid.UUID, len(userTaskList))
		taskIDs := make([]uuid.UUID, len(userTaskList))
		userIDs := make([]uuid.UUID, len(userTaskList))
		roles := make([]string, len(userTaskList))

		for i, ut := range userTaskList {
			ids[i] = ut.Id
			taskIDs[i] = task.Id
			userIDs[i] = ut.UserId
			roles[i] = string(ut.Role)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO user_task (id, task_id, user_id, role)
			 SELECT 
				unnest($1::uuid[]),
				unnest($2::uuid[]),
				unnest($3::uuid[]),
				unnest($4::text[])`,
			ids, taskIDs, userIDs, roles,
		)
		if err != nil {
			return fmt.Errorf("failed to save user tasks: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (d *DBStorage) GetOne(subscriprionID int) (model.Subscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var subscription model.Subscription

	err := d.pool.QueryRow(ctx,
		`SELECT service_name, price, user_id, start_date FROM subscriptions WHERE id = $1`,
		subscriprionID).Scan(&subscription.ServiceName, &subscription.Price, &subscription.UserID, &subscription.StartDate)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Subscription{}, ErrSubscriptionNotFound
		}
		return model.Subscription{}, err
	}
	return subscription, nil
}

func (d *DBStorage) GetList() ([]model.Subscription, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT service_name, price, user_id, start_date FROM subscriptions ORDER BY start_date DESC`)
	if err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}
	defer rows.Close()

	var subscriptionsList []model.Subscription
	for rows.Next() {
		var subscription model.Subscription
		if err := rows.Scan(&subscription.ServiceName, &subscription.Price, &subscription.UserID, &subscription.StartDate); err != nil {
			return nil, fmt.Errorf("data scan error: %w", err)
		}
		subscriptionsList = append(subscriptionsList, subscription)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows processing error: %w", err)
	}

	return subscriptionsList, nil
}

func (d *DBStorage) Update(subscriptionID int, subscription model.Subscription) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := d.pool.Exec(ctx,
		`UPDATE subscriptions SET service_name = $1, price = $2, user_id = $3, start_date = $4 WHERE id = $5`,
		subscription.ServiceName, subscription.Price, subscription.UserID, subscription.StartDate, subscriptionID)

	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func (d *DBStorage) Delete(subscriptionID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := d.pool.Exec(ctx,
		`DELETE FROM subscriptions WHERE id = $1`,
		subscriptionID,
	)

	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func (d *DBStorage) GetSum(serviceName string, userID uuid.UUID, startPeriod time.Time, endPeriod time.Time) (model.TotalPriceResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var totalPrice int

	err := d.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(price), 0)
         FROM subscriptions
         WHERE service_name = $1
           AND user_id = $2
           AND start_date >= $3
           AND start_date <= $4`,
		serviceName, userID, startPeriod, endPeriod,
	).Scan(&totalPrice)

	if err != nil {
		return model.TotalPriceResponse{}, fmt.Errorf("failed to get sum: %w", err)
	}

	return model.TotalPriceResponse{TotalPrice: totalPrice}, nil
}
