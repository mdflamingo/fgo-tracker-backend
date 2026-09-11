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
				unnest($4::task_role[])`,
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

	rows, err := d.pool.Query(ctx,
		`SELECT
			t.id, t.name, t.description, t.status, t.priority, t.deadline, t.completed_at,
			p.id, p.name,
			u.id, u.username, u.email,
			ut.role
		FROM task t
		LEFT JOIN project p ON t.project_id = p.id
		LEFT JOIN user_task ut ON t.id = ut.task_id
		LEFT JOIN "user" u ON ut.user_id = u.id
		WHERE t.id = $1`,
		taskID)
	if err != nil {
		return model.TaskResponse{}, err
	}
	defer rows.Close()

	var task model.TaskResponse
	var taskFound bool

	creatorSet := make(map[uuid.UUID]bool)
	reviewerSeen := make(map[uuid.UUID]bool)
	assigneeSeen := make(map[uuid.UUID]bool)

	for rows.Next() {
		var (
			tID       uuid.UUID
			name      string
			desc      string
			status    model.TaskStatus
			priority  model.TaskPriority
			deadline  *time.Time
			completed *time.Time
			pID       *uuid.UUID
			pName     *string
			uID       *uuid.UUID
			uName     *string
			uEmail    *string
			role      *string
		)

		if err := rows.Scan(
			&tID, &name, &desc, &status, &priority, &deadline, &completed,
			&pID, &pName,
			&uID, &uName, &uEmail,
			&role,
		); err != nil {
			return model.TaskResponse{}, err
		}

		if !taskFound {
			task = model.TaskResponse{
				Id:          tID,
				Name:        name,
				Description: desc,
				Status:      status,
				Priority:    priority,
				Deadline:    deadline,
				Completed:   completed,
			}
			if pID != nil && pName != nil {
				task.Project = model.ProjectDB{Id: *pID, Name: *pName}
			}
			taskFound = true
		}

		if uID == nil || role == nil {
			continue
		}

		user := model.UserDB{
			Id:       *uID,
			Username: derefStr(uName),
			Email:    derefStr(uEmail),
		}

		switch *role {
		case string(model.TaskCreator):
			if !creatorSet[*uID] {
				task.Creator = user
				creatorSet[*uID] = true
			}
		case string(model.TaskReviewer):
			if !reviewerSeen[*uID] {
				task.Reviewers = append(task.Reviewers, user)
				reviewerSeen[*uID] = true
			}
		case string(model.TaskAssignee):
			if !assigneeSeen[*uID] {
				task.Assignees = append(task.Assignees, user)
				assigneeSeen[*uID] = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		return model.TaskResponse{}, err
	}

	if !taskFound {
		return model.TaskResponse{}, ErrTaskNotFound
	}

	return task, nil
}

func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func (d *DBStorage) GetList() ([]model.TaskListResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT t.id, t.name, t.description, t.status, t.priority, p.name
			FROM task t
			LEFT JOIN project p ON t.project_id = p.id
			ORDER BY t.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}
	defer rows.Close()

	var tasksList []model.TaskListResponse
	for rows.Next() {
		var task model.TaskListResponse
		if err := rows.Scan(&task.Id, &task.Name, &task.Description, &task.Status, &task.Priority, &task.ProjectName); err != nil {
			return nil, fmt.Errorf("data scan error: %w", err)
		}
		tasksList = append(tasksList, task)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows processing error: %w", err)
	}

	return tasksList, nil
}

func (d *DBStorage) Update(taskID uuid.UUID, task model.TaskUpdate, userTasks []model.TaskUserCreate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE task
			SET name = $1, description = $2, status = $3, priority = $4, project_id = $5, deadline = $6, completed_at = $7
		WHERE id = $8`,
		task.Name, task.Description, task.Status, task.Priority, task.ProjectId, task.Deadline, task.CompletedAt, taskID)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	_, err = tx.Exec(ctx, `DELETE FROM user_task WHERE task_id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete user tasks: %w", err)
	}

	if len(userTasks) > 0 {
		ids := make([]uuid.UUID, len(userTasks))
		taskIDs := make([]uuid.UUID, len(userTasks))
		userIDs := make([]uuid.UUID, len(userTasks))
		roles := make([]string, len(userTasks))

		for i, ut := range userTasks {
			ids[i] = ut.Id
			taskIDs[i] = taskID
			userIDs[i] = ut.UserId
			roles[i] = string(ut.Role)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO user_task (id, task_id, user_id, role)
			 SELECT
				unnest($1::uuid[]),
				unnest($2::uuid[]),
				unnest($3::uuid[]),
				unnest($4::task_role[])`,
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

func (d *DBStorage) Delete(taskID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := d.pool.Exec(ctx,
		`DELETE FROM task WHERE id = $1`,
		taskID,
	)

	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}
