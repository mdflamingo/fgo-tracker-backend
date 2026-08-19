-- ==============================================================================
-- ОТКАТ МИГРАЦИИ (DOWN MIGRATION)
-- Выполняется в строго обратном порядке для соблюдения зависимостей
-- ==============================================================================

-- 1. УДАЛЕНИЕ JUNCTION ТАБЛИЦ (СВЯЗЕЙ)
-- Удаляем их первыми, так как они зависят от доменных таблиц
DROP TABLE IF EXISTS user_task;
DROP TABLE IF EXISTS user_project;

-- 2. УДАЛЕНИЕ ТАБЛИЦЫ ВЛОЖЕНИЙ
-- Зависит от task и "user"
DROP TABLE IF EXISTS attachment;

-- 3. УДАЛЕНИЕ ДОМЕННЫХ ТАБЛИЦ (В ОБРАТНОМ ПОРЯДКЕ ЗАВИСИМОСТЕЙ)
-- task зависит от project
DROP TABLE IF EXISTS task;

-- project и "user" ни от кого не зависят (на этом этапе)
DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS "user";

-- 4. УДАЛЕНИЕ ТИПОВ (ENUM)
-- Удаляем только после того, как удалены все таблицы, которые их использовали
DROP TYPE IF EXISTS attachment_type;
DROP TYPE IF EXISTS project_role;
DROP TYPE IF EXISTS task_role;
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_status;

-- 5. УДАЛЕНИЕ ВСПОМОГАТЕЛЬНОЙ ФУНКЦИИ
-- Триггеры, которые её вызывали, уже удалены вместе с таблицами
DROP FUNCTION IF EXISTS update_updated_at_column();