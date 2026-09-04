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
	ErrTaskNotFound = errors.New("task not found")
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

func (d *DBStorage) GetOne(taskID uuid.UUID) (model.TaskResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var task model.TaskResponse

	err := d.pool.QueryRow(ctx,
		`SELECT
			t.id,
			t.name,
			t.description,
			t.status,
			t.priority,
			t.deadline,
			t.completed_at,
			json_build_object('id', p.id, 'name', p.name) as project,
			(
				SELECT jsonb_build_object('id', u2.id, 'username', u2.username, 'email', u2.email)
				FROM user_task ut2 JOIN "user" u2 ON ut2.user_id = u2.id
				WHERE ut2.task_id = t.id AND ut2.role = 'creator'
				LIMIT 1
			) AS creator,
			COALESCE(jsonb_agg(jsonb_build_object('id', u.id, 'username', u.username, 'email', u.email)) FILTER (WHERE u.id IS NOT NULL AND ut.role = 'reviewer'), '[]'::jsonb)
			AS reviewers,
			COALESCE(jsonb_agg(jsonb_build_object('id', u.id, 'username', u.username, 'email', u.email)) FILTER (WHERE u.id IS NOT NULL AND ut.role = 'assignee'), '[]'::jsonb)
			AS assignees
		FROM task t
		LEFT JOIN project p ON t.project_id = p.id
		LEFT JOIN user_task ut ON t.id = ut.task_id
		LEFT JOIN "user" u ON ut.user_id = u.id
		WHERE t.id = $1
		GROUP BY t.id, t.name, t.description, t.status, t.priority, t.deadline, t.completed_at, p.id, p.name;`,
		taskID).Scan(&task.Id, &task.Name, &task.Description, &task.Status, &task.Priority, &task.Deadline, &task.Completed, &task.Project, &task.Creator, &task.Reviewer, &task.Assigned)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.TaskResponse{}, ErrTaskNotFound
		}
		return model.TaskResponse{}, err
	}
	return task, nil
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
