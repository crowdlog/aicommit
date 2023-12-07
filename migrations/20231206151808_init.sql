-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_settings (
    id TEXT PRIMARY KEY, 
    model_selection TEXT
);

CREATE INDEX idx_model_selection ON user_settings(model_selection);

CREATE TABLE commits (
    id INTEGER PRIMARY KEY,
    git_diff TEXT,
    commit_message TEXT, 
    git_diff_command TEXT
);
-- +goose StatementEnd

-- +goose Down 
-- +goose StatementBegin
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS commits;
-- +goose StatementEnd