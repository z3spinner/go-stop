-- name: GetSetting :one
SELECT value FROM app_settings WHERE key = $1;

-- name: InsertSettingIfAbsent :exec
INSERT INTO app_settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO NOTHING;
