-- +goose Up
-- +goose StatementBegin
update staff_templates
set description = 'Your bot''s invite link requests the Administrator permission, which Omniplex does not accept. Per the Bot Rules, a bot must only request the permissions its commands actually use. A kick command only needs Kick Members, for example. Paste your invite link into https://noadmin.info/analyze to see everything it asks for, then build a new link with only what your bot needs at https://noadmin.info/calculator. The guide at https://noadmin.info/guides/why-not-administrator explains why this matters. Update your invite link and resubmit.'
where name = 'Bot: Requires Administrator Permission'
  and entity_type = 'bot'
  and description = 'Your bot requires the Administrator permission to function. Per the Bot Rules, commands must only request the specific permissions they actually need — a kick command should only need Kick Members, for example. Please scope your bot''s permissions down and resubmit.';

insert into staff_templates (name, emoji, tags, description, type, entity_type)
select v.name, v.emoji, v.tags, v.description, v.type, v.entity_type
from (values
  ('Bot: Excessive Permissions', '🛡️', array['denial', 'permissions'],
   'Your bot''s invite link does not request Administrator, but it requests high-risk permissions that none of your bot''s commands use. Per the Bot Rules, a bot must only request the permissions it actually needs. Paste your invite link into https://noadmin.info/analyze to see which permissions are high risk, and compare it with the examples at https://noadmin.info/examples. Remove what your bot does not use and resubmit.',
   'denial', 'bot'),
  ('Bot: Broken Invite Link', '🔗', array['denial', 'availability'],
   'Your bot''s invite link does not work. It may have the wrong client ID, an invalid permissions value, or be missing the bot scope. You can build a working link at https://noadmin.info/calculator. Update your invite link and resubmit.',
   'denial', 'bot')
) as v(name, emoji, tags, description, type, entity_type)
where not exists (
  select 1 from staff_templates t where t.name = v.name and t.entity_type = v.entity_type
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
delete from staff_templates
where entity_type = 'bot'
  and name in ('Bot: Excessive Permissions', 'Bot: Broken Invite Link');

update staff_templates
set description = 'Your bot requires the Administrator permission to function. Per the Bot Rules, commands must only request the specific permissions they actually need — a kick command should only need Kick Members, for example. Please scope your bot''s permissions down and resubmit.'
where name = 'Bot: Requires Administrator Permission'
  and entity_type = 'bot'
  and description like 'Your bot''s invite link requests the Administrator permission%';
-- +goose StatementEnd
