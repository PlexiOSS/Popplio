-- Panel-only queries with no other natural home (arcadia/panel ops_content.go).

-- name: CountPartnerTypeByID :one
SELECT EXISTS(SELECT 1 FROM partner_types WHERE id = $1);

-- name: ListPartners :many
SELECT id, name, short, links, type, created_at, user_id, bot_id FROM partners;

-- name: ListPartnerTypes :many
SELECT id, name, short, icon, created_at FROM partner_types;

-- name: CountPartnerByID :one
SELECT EXISTS(SELECT 1 FROM partners WHERE id = $1);

-- name: InsertPartner :exec
INSERT INTO partners (id, name, short, links, type, user_id, bot_id) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdatePartner :exec
UPDATE partners SET name = $2, short = $3, links = $4, type = $5, user_id = $6, bot_id = $7 WHERE id = $1;

-- name: DeletePartner :exec
DELETE FROM partners WHERE id = $1;
