-- ==============================================================================
-- 1. ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
-- ==============================================================================

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ==============================================================================
-- 2. СОЗДАНИЕ ТИПОВ (ENUM)
-- ==============================================================================

CREATE TYPE task_status AS ENUM ('backlog', 'in_progress', 'review', 'done');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high');
CREATE TYPE task_role AS ENUM ('creator', 'assignee', 'reviewer');
CREATE TYPE project_role AS ENUM ('owner', 'member', 'viewer');
CREATE TYPE attachment_type AS ENUM ('image', 'file', 'link');

-- ==============================================================================
-- 3. ДОМЕННЫЕ ТАБЛИЦЫ
-- ==============================================================================

-- Таблица USER
CREATE TABLE "user" (
    id uuid PRIMARY KEY,
    username varchar(256) NOT NULL,
    email varchar(320) NOT NULL,
    deleted_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT uq_user_username UNIQUE (username)
);

-- Уникальный регистронезависимый индекс для email
CREATE UNIQUE INDEX idx_user_email_lower ON "user" (lower(email));

-- Триггер для updated_at
CREATE TRIGGER trg_user_updated_at
    BEFORE UPDATE ON "user"
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- Таблица PROJECT
CREATE TABLE project (
    id uuid PRIMARY KEY,
    name varchar(256) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_project_name_length CHECK (char_length(name) BETWEEN 1 AND 256)
);

CREATE TRIGGER trg_project_updated_at
    BEFORE UPDATE ON project
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- Таблица TASK
CREATE TABLE task (
    id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    name varchar(255) NOT NULL,
    description text NULL,
    status task_status NOT NULL DEFAULT 'backlog',
    priority task_priority NOT NULL DEFAULT 'low',
    deadline timestamptz NULL,
    completed_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_task_name_length CHECK (char_length(name) BETWEEN 1 AND 255),
    CONSTRAINT fk_task_project FOREIGN KEY (project_id) 
        REFERENCES project (id) ON DELETE CASCADE
);

-- Индекс для быстрого поиска задач по проекту
CREATE INDEX idx_task_project_id ON task (project_id);

CREATE TRIGGER trg_task_updated_at
    BEFORE UPDATE ON task
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ==============================================================================
-- 4. ТАБЛИЦА ВЛОЖЕНИЙ
-- ==============================================================================

CREATE TABLE attachment (
    id uuid PRIMARY KEY,
    task_id uuid NOT NULL,
    file_type attachment_type NOT NULL,
    link_url text NULL,
    storage_path text NULL,
    original_name text NULL,
    mime_type varchar(128) NULL,
    size_bytes bigint NULL,
    creator_id uuid NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_attachment_size CHECK (size_bytes IS NULL OR size_bytes > 0),
    CONSTRAINT fk_attachment_task FOREIGN KEY (task_id) 
        REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_attachment_creator FOREIGN KEY (creator_id) 
        REFERENCES "user" (id) ON DELETE SET NULL
);

CREATE INDEX idx_attachment_task_id ON attachment (task_id);
CREATE INDEX idx_attachment_creator_id ON attachment (creator_id);

-- ==============================================================================
-- 5. JUNCTION ТАБЛИЦЫ (СВЯЗИ)
-- ==============================================================================

-- Связь USER <-> PROJECT
CREATE TABLE user_project (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    project_id uuid NOT NULL,
    role project_role NOT NULL DEFAULT 'member',
    created_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT fk_user_project_user FOREIGN KEY (user_id) 
        REFERENCES "user" (id) ON DELETE CASCADE,
    CONSTRAINT fk_user_project_project FOREIGN KEY (project_id) 
        REFERENCES project (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_project_user_id ON user_project (user_id);
CREATE INDEX idx_user_project_project_id ON user_project (project_id);


-- Связь USER <-> TASK
CREATE TABLE user_task (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    task_id uuid NOT NULL,
    role task_role NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT fk_user_task_user FOREIGN KEY (user_id) 
        REFERENCES "user" (id) ON DELETE CASCADE,
    CONSTRAINT fk_user_task_task FOREIGN KEY (task_id) 
        REFERENCES task (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_task_user_id ON user_task (user_id);
CREATE INDEX idx_user_task_task_id ON user_task (task_id);