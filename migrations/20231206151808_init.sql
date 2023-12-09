-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_settings (
    id TEXT PRIMARY KEY, 
    model_selection TEXT,
    exclude_files TEXT,
    use_conventional_commits BOOLEAN,
    date_created TIMESTAMP
);

CREATE INDEX idx_model_selection ON user_settings(model_selection);

CREATE TABLE commits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    commit_message TEXT, 
    git_diff_command TEXT,
    git_diff_command_output TEXT,
    exclude_files TEXT,
    date_created TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down 
-- +goose StatementBegin
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS commits;
-- +goose StatementEnd