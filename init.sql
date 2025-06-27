-- Create the tasks table with PostgreSQL syntax
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    base_url VARCHAR(255) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(255) NOT NULL,
    headers TEXT NOT NULL,
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'resolved')),
    response TEXT DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


-- Create a function to notify on pending task insertion
CREATE OR REPLACE FUNCTION notify_pending_task()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'pending' THEN
        PERFORM pg_notify('pending_task', NEW.id::text);
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create a trigger to notify when pending tasks are inserted
CREATE TRIGGER pending_task_notification
    AFTER INSERT ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION notify_pending_task();
