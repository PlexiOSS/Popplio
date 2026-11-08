-- name: GetSitePrefs :one
SELECT prefs, updated_at FROM user_site_prefs WHERE user_id = $1;

-- name: UpsertSitePrefs :one
INSERT INTO user_site_prefs (user_id, prefs) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET prefs = EXCLUDED.prefs, updated_at = NOW()
RETURNING updated_at;
