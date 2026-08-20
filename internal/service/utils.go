package service

import (
	"database/sql/driver"
	"fmt"
)

// TaskStatus представляет статус задачи.
// Базовый тип int позволяет принимать числа из JSON (как IntEnum в Python).
type TaskStatus int

// Константы. Начинаем с 1, чтобы 0 остался для невалидного/неинициализированного состояния.
const (
	TaskStatusUnknown    TaskStatus = 0
	TaskStatusBacklog    TaskStatus = 1
	TaskStatusInProgress TaskStatus = 2
	TaskStatusReview     TaskStatus = 3
	TaskStatusDone       TaskStatus = 4
)

// Маппинг Go int -> PostgreSQL string (для записи в БД)
var taskStatusToDB = map[TaskStatus]string{
	TaskStatusBacklog:    "backlog",
	TaskStatusInProgress: "in_progress",
	TaskStatusReview:     "review",
	TaskStatusDone:       "done",
}

// Маппинг PostgreSQL string -> Go int (для чтения из БД)
var dbToTaskStatus = map[string]TaskStatus{
	"backlog":     TaskStatusBacklog,
	"in_progress": TaskStatusInProgress,
	"review":      TaskStatusReview,
	"done":        TaskStatusDone,
}

// Value реализует интерфейс driver.Valuer.
// Нужен, чтобы драйвер БД знал, как записать наш int в PostgreSQL ENUM (строку).
func (s TaskStatus) Value() (driver.Value, error) {
	if str, ok := taskStatusToDB[s]; ok {
		return str, nil
	}
	return nil, fmt.Errorf("invalid task status: %d", s)
}

// Scan реализует интерфейс sql.Scanner.
// Нужен, чтобы драйвер БД знал, как прочитать строку из PostgreSQL ENUM обратно в наш int.
func (s *TaskStatus) Scan(src interface{}) error {
	if src == nil {
		*s = TaskStatusUnknown
		return nil
	}

	// Получаем строку из интерфейса (зависит от драйвера, может быть string или []byte)
	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("could not scan task status: unexpected type %T", src)
	}

	if val, ok := dbToTaskStatus[str]; ok {
		*s = val
		return nil
	}
	return fmt.Errorf("invalid task status from db: %s", str)
}

// IsValid проверяет, что статус валиден (не 0 и не мусор).
func (s TaskStatus) IsValid() bool {
	_, ok := taskStatusToDB[s]
	return ok
}

// String для красивого логирования.
func (s TaskStatus) String() string {
	if str, ok := taskStatusToDB[s]; ok {
		return str
	}
	return fmt.Sprintf("unknown(%d)", s)
}
