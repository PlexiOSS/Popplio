-- Servers currently default to type = 'approved' at insert time (bots default
-- to 'pending'), so every server added via PUT /servers has gone live
-- immediately with zero staff review, unlike bots. This is metadata-only on
-- Postgres 11+ (no rewrite) and only affects future inserts — existing rows
-- keep whatever type they already have.
alter table servers alter column type set default 'pending';

-- Bots already carry a staff note through their review (bots.approval_note,
-- NOT NULL DEFAULT ''). Servers have no equivalent column, needed now that
-- Claim/Approve/Deny/Unverify support Server as a target type — matching
-- bots' NOT NULL DEFAULT '' here, not just a bare nullable column: the Go
-- side (searchServerRow.ApprovalNote) scans this into a plain string, same
-- as bots' row struct does, and a NULL there fails that scan outright.
alter table servers add column approval_note text not null default '';
