-- name: GetWordBlacklistSystems :one
SELECT systems FROM blacklisted_words WHERE word = $1;
