-- name: GetDiscordUsersDueForRefresh :many
SELECT iuc.id FROM internal_user_cache__discord iuc
WHERE iuc.last_updated < NOW() - make_interval(secs => sqlc.arg(min_age_seconds)::double precision)
ORDER BY
    CASE
        WHEN EXISTS (SELECT 1 FROM bots b WHERE b.bot_id = iuc.id) THEN 0
        WHEN EXISTS (SELECT 1 FROM bots b WHERE b.owner = iuc.id)
          OR EXISTS (SELECT 1 FROM team_members tm WHERE tm.user_id = iuc.id) THEN 1
        ELSE 2
    END,
    iuc.last_updated ASC
LIMIT sqlc.arg(batch_size);
