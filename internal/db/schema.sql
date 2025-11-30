-- Database schema for JiraF bot

-- Create chats table
CREATE TABLE IF NOT EXISTS chats (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create chat_settings table
CREATE TABLE IF NOT EXISTS chat_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(id),
    todoist_project_id TEXT,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(id),
    owner_id BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS sessions_chat_id_idx ON sessions(chat_id);
CREATE INDEX IF NOT EXISTS sessions_status_idx ON sessions(status);

-- Create messages table
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(id),
    session_id INTEGER REFERENCES sessions(id),
    message_id INTEGER NOT NULL,
    user_id BIGINT,
    username TEXT,
    text TEXT,
    ts TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS messages_chat_id_idx ON messages(chat_id);
CREATE INDEX IF NOT EXISTS messages_session_id_idx ON messages(session_id);
CREATE INDEX IF NOT EXISTS messages_ts_idx ON messages(ts);

-- Create draft_tasks table
CREATE TABLE IF NOT EXISTS draft_tasks (
    session_id INTEGER PRIMARY KEY REFERENCES sessions(id),
    title TEXT,
    description TEXT,
    due_iso TEXT,
    priority INTEGER,
    assignee_note TEXT,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create created_tasks table
CREATE TABLE IF NOT EXISTS created_tasks (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES sessions(id),
    todoist_task_id TEXT NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS created_tasks_session_id_idx ON created_tasks(session_id);

-- Create audit_edits table
CREATE TABLE IF NOT EXISTS audit_edits (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES sessions(id),
    instruction_text TEXT NOT NULL,
    diff_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS audit_edits_session_id_idx ON audit_edits(session_id);