create index entity_votes_target_created_idx on entity_votes(target_type, target_id, created_at) where void = false;
