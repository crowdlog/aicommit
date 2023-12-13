-- +goose Up
-- +goose StatementBegin
CREATE TABLE
  user_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ai_provider TEXT,
    model_selection TEXT,
    exclude_files TEXT,
    use_conventional_commits BOOLEAN,
    date_created TIMESTAMP
  );

CREATE TABLE
  commits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    commit_message TEXT,
    git_diff_command TEXT,
    git_diff_command_output TEXT,
    exclude_files TEXT,
    date_created TIMESTAMP
  );

CREATE TABLE
  diff (
    id TEXT PRIMARY KEY,
    diff TEXT,
    date_created TIMESTAMP,
    diff_structured_json TEXT,
    model TEXT,
    ai_provider TEXT,
    prompts TEXT
  );
-- +goose StatementEnd
-- +goose Down 
-- +goose StatementBegin
DROP TABLE IF EXISTS user_settings;

DROP TABLE IF EXISTS commits;

-- +goose StatementEnd