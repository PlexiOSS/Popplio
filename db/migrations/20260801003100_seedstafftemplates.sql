-- +goose Up
-- +goose StatementBegin
-- Initial set of staff review templates — pre-built answers for the
-- approve/deny flow, split by entity_type (bot vs server) and grounded in
-- the actual published rules (see documentation/content/docs/guides/rules/).
-- Not exhaustive; add more via /admin/templates as real review patterns
-- come up. `type` is a loose category ("approval"/"denial") for grouping in
-- the picker UI, not enforced anywhere server-side.

-- === Bots ===

insert into staff_templates (name, emoji, tags, description, type, entity_type) values
('Bot: Approved', '✅', array['approval'],
 'Congratulations, your bot has been approved and is now live on Omniplex! Thanks for following the listing rules — if you ever add major new features, feel free to update your page to reflect them.',
 'approval', 'bot'),

('Bot: Offline During Review', '🔴', array['denial', 'availability'],
 'Your bot was offline (or not public/invitable) at the time of review. Per the Bot Rules, your bot must be online and invitable during review. Please make sure it''s running and resubmit.',
 'denial', 'bot'),

('Bot: Not Enough Working Commands', '🧩', array['denial', 'functionality'],
 'Your bot doesn''t currently meet the minimum of 7 working commands (or doesn''t have a clear point of entry, like a working help command), unless it serves one single designated purpose. Please add more functionality or fix what''s broken and resubmit.',
 'denial', 'bot'),

('Bot: Broken Commands', '🐛', array['denial', 'functionality'],
 'One or more of your bot''s commands aren''t working correctly during review. Please fix and thoroughly test your commands before resubmitting.',
 'denial', 'bot'),

('Bot: Requires Administrator Permission', '🔑', array['denial', 'permissions'],
 'Your bot requires the Administrator permission to function. Per the Bot Rules, commands must only request the specific permissions they actually need — a kick command should only need Kick Members, for example. Please scope your bot''s permissions down and resubmit.',
 'denial', 'bot'),

('Bot: Unmodified Fork/Instance', '🍴', array['denial', 'originality'],
 'Your bot appears to be an unmodified (or lightly modified) instance/fork of another bot. Per the Bot Rules, forks need considerable modification or original additions to be listed. Please add meaningful original functionality and resubmit.',
 'denial', 'bot'),

('Bot: Unauthorized YouTube/Music Streaming', '🎵', array['denial', 'tos'],
 'Bots that allow downloading, streaming, or sharing music via YouTube violate YouTube''s, Discord''s, and Omniplex''s Terms of Service, and cannot be listed. This isn''t something we can make an exception for.',
 'denial', 'bot'),

('Bot: Padded/Repeated Commands', '📋', array['denial', 'quality'],
 'Your bot appears to repeat the same command with only minor variations to pad out its command count, rather than offering genuinely distinct functionality. Please consolidate or add real functionality and resubmit.',
 'denial', 'bot'),

('Bot: Impersonation', '⚠️', array['denial', 'impersonation'],
 'Your bot appears to intentionally impersonate another listed bot (name, avatar, and/or description). Bots may not have the sole intent of impersonating another bot. Please differentiate your bot''s identity and resubmit.',
 'denial', 'bot'),

('Bot: Owner Commands Not Locked', '🔒', array['denial', 'security'],
 'Sensitive owner/developer-only commands (evals, status/presence changes, etc.) aren''t properly locked to owners/developers. Please restrict access to these commands before resubmitting.',
 'denial', 'bot');

-- === Servers ===

insert into staff_templates (name, emoji, tags, description, type, entity_type) values
('Server: Approved', '✅', array['approval'],
 'Congratulations, your server has been approved and is now live on Omniplex! Thanks for following the listing rules.',
 'approval', 'server'),

('Server: Invalid or Expired Invite', '🔗', array['denial', 'availability'],
 'Your server''s invite link is expired, one-time-use, or otherwise not working. Per the Server Listing Rules, your server must have a working, permanent invite link at all times. Please provide a permanent invite and resubmit.',
 'denial', 'server'),

('Server: Misrepresented Activity/Member Count', '📊', array['denial', 'integrity'],
 'Your server''s listing doesn''t accurately represent its actual activity or member count. Listings must not misrepresent activity, purpose, or content to inflate rankings. Please correct your listing and resubmit.',
 'denial', 'server'),

('Server: NSFW Content Not Gated', '🔞', array['denial', 'nsfw'],
 'Your server includes NSFW content or imagery that isn''t properly restricted to age-gated NSFW channels, or your server isn''t correctly tagged as NSFW. Please fix your channel gating and/or tags and resubmit.',
 'denial', 'server'),

('Server: Vote/Invite Farming', '🚫', array['denial', 'manipulation'],
 'Your server appears to exist solely to farm votes, invites, or rewards without providing genuine community value, or uses deceptive invite-reward schemes to inflate its numbers. This isn''t something we can approve as-is.',
 'denial', 'server'),

('Server: Impersonation', '⚠️', array['denial', 'impersonation'],
 'Your server appears to intentionally impersonate another server or brand. Please differentiate your server''s identity (name, icon, description) and resubmit.',
 'denial', 'server'),

('Server: Malicious or Unrelated Extra Links', '🔗', array['denial', 'links'],
 'One or more of your server''s Extra Links point to unrelated or malicious content rather than real resources (website, socials, support). Please update your links to point to legitimate destinations and resubmit.',
 'denial', 'server');
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/seedstafftemplates.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
