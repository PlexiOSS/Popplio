-- +goose Up
-- +goose StatementBegin
CREATE TABLE app_positions (
    id text PRIMARY KEY,
    name text NOT NULL,
    info text NOT NULL DEFAULT '',
    tags text[] NOT NULL DEFAULT '{}',
    questions jsonb NOT NULL DEFAULT '[]',
    hidden boolean NOT NULL DEFAULT false,
    closed boolean NOT NULL DEFAULT false,
    requires_main_server boolean NOT NULL DEFAULT false,
    cooldown_secs bigint NOT NULL DEFAULT 0,
    approval_note text NOT NULL DEFAULT '',
    help_url text,
    help_label text,
    behaviour text,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$staff$v$, $v$Staff Team$v$, $v$Join the Omniplex Staff Team and help us Approve, Deny and Certify Discord Bots.

We are a welcoming and laid back team who is always willing to give new people an opportunity!$v$, ARRAY[$v$Staff$v$, $v$Bot Reviewing$v$]::text[], $v$[{"id":"experience","question":"Past server experience","paragraph":"Tell us any experience you have working for other servers or bot lists.","placeholder":"I have worked at...","short":false},{"id":"strengths","question":"List some of your strengths","paragraph":"What are your strengths/abilities. How long can you be active?","placeholder":"I am always online and active...","short":false},{"id":"situations","question":"Situation Examples","paragraph":"How would you handle: Mass Pings, Nukes and Raids etc.","placeholder":"I would handle it by...","short":false},{"id":"reason","question":"Why do you want to join the staff team?","paragraph":"Why do you want to join the staff team? Be specific","placeholder":"I want to join the staff team because...","short":false},{"id":"team-player","question":"What is a scenario in which you had to be a team player?","paragraph":"What is a scenario in which you had to be a team player? We want to know that you can collaborate effectively with us.","placeholder":"I had to...","short":false},{"id":"about-you","question":"Tell us a little about yourself","paragraph":"Tell us a little about yourself. Its that simple!","placeholder":"I am...","short":false},{"id":"other","question":"Anything else you want to add?","paragraph":"Anything else you want to add?","placeholder":"Just state anything that doesn't hit anywhere else","short":true}]$v$::jsonb, false, false, false, 172800, $v$Next, join our Discord server at https://discord.gg/AxrR4RNSFT if you haven't already, then ping a manager in #staff-only to start onboarding. If you can't see that channel yet, ask any staff member.$v$, $v$staff$v$, 0);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$dev$v$, $v$Dev Team$v$, $v$Join our Dev Team and help us update, manage and maintain all of the Omniplex Services!.

Some experience in PostgreSQL and at least one of the below languages is required:

- TypeScript (Javascript with type-safety)
- Go/Golang$v$, ARRAY[$v$Golang$v$, $v$Rust$v$]::text[], $v$[{"id":"sql-basics-1","question":"Write a SQL expression to select from a table named 'shop' the columns price (float) and quantity (integer) limited to 6 rows, ordered by the price in descending order","paragraph":"Answer the questions above using the most readable and (where possible) the most optimized SQL. Assume PostgreSQL 15 is being used and the 'pgxpool' (go) driver is being used.","placeholder":"SQL Here","short":false},{"id":"sql-basics-2","question":"You now need to select all rows of the 'shop' table where rating (float) is above 5, the name (text) is similar (and case-insensitive) to $1 and categories (text[]) contains at least one element from $2 and all elements of $3 where $1, $2 and $3 are parameters of a parameterized query","paragraph":"Answer the questions above using the most readable and (where possible) the most optimized SQL. Assume PostgreSQL 15 is being used and the 'pgxpool' (go) driver is being used.","placeholder":"SQL Here","short":false},{"id":"foobar","question":"Write a program that loops over all numbers from 1 to 7847 (inclusive). For every multiple of 7 and not 19, print 7 times the number and a uppercase A (on the same line), for every multiple of 19 and not 7, print a lowercase B and 5 more than the number divided by 4 and rounded (on the same line), for every multiple of both 7 and 19 print 'foobar'. Otherwise print 24 times the number","paragraph":"Answer the question above with the least amount of code. Use either Go 1.18 for all solutions. Your solution must NOT link to an external resource or library and you MUST justify all code with comments","placeholder":"Code here...","short":false},{"id":"experience","question":"Do you have experience in Typescript and/or Go. Give examples of projects/code you have written","paragraph":"Do you have experience in Typescript, Rust and/or Go. Give examples of projects/code you have written.","placeholder":"I have worked on...","short":false},{"id":"db","question":"Do you have Exprience with PostgreSQL. How much experience do you have?","paragraph":"Do you have Exprience with PostgreSQL","placeholder":"I have used PostgreSQL for... and know...","short":false},{"id":"reason","question":"Why do you want to join the dev team?","paragraph":"Why do you want to join the dev team? Be specific","placeholder":"I want to join the dev team because...","short":false},{"id":"team-player","question":"What is a scenario in which you had to be a team player?","paragraph":"What is a scenario in which you had to be a team player? We want to know that you can collaborate effectively with us.","placeholder":"I had to...","short":false},{"id":"other","question":"Anything else you want to add?","paragraph":"Anything else you want to add?","placeholder":"Just state anything that doesn't hit anywhere else","short":true}]$v$::jsonb, false, false, false, 172800, $v$$v$, NULL, 1);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$partners$v$, $v$Partners$v$, $v$Partner your Discord Bot, Discord Server or Business today! It's easier than ever before!

Some points to note:

- When you apply for a partnership, make sure that you are authorized to speak on the services behalf
- NodeByte LTD reserves the right to deny or cancel any partnership application at any time.
$v$, ARRAY[$v$Advertising$v$, $v$Business$v$]::text[], $v$[{"id":"what","question":"What are you looking to partner with us for?","paragraph":"What are you looking to partner with us for? Be descriptive here","placeholder":"I wish to partner a bot/website called Foobar because...","short":true},{"id":"why","question":"Why do you want to partner with us?","paragraph":"Why do you want to partner with us? Be specific","placeholder":"I want to partner with Omniplex because...","short":true},{"id":"how","question":"How will you promote us?","paragraph":"How will you promote Omniplex? This could be a partner command or a link on your website!","placeholder":"I will promote Omniplex using...","short":true},{"id":"demo","question":"Do you have anything to showcase what you wish to partner with us?","paragraph":"Links to show us demos of what you're partnering or how many members your server or bot has.","placeholder":"LINK 1 etc.","short":false},{"id":"other","question":"Anything else you want to add?","paragraph":"Anything else you want to add?","placeholder":"Just state anything that doesn't hit anywhere else","short":true}]$v$::jsonb, false, false, true, 0, $v$$v$, NULL, 2);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$resubmit$v$, $v$Bot Resubmission$v$, $v$Resubmit your denied bot to the list!$v$, ARRAY[$v$Resubmissions$v$]::text[], $v$[{"id":"id","question":"Bot ID?","paragraph":"What is the bot ID?","placeholder":"Bot ID","short":true},{"id":"reason","question":"Anything else?","paragraph":"Make sure you know why your bot was denied and that you have fixed the problem. If you don't know why your bot was denied, please contact us on Discord","placeholder":"I believe.../I fixed.../The bot was offline because...","short":true}]$v$::jsonb, true, false, false, 0, $v$$v$, $v$resubmit$v$, 3);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$banappeal$v$, $v$Ban Appeal$v$, $v$<h3 class="text-2xl font-semibold">Hello There, Welcome</h3>
If you find yourself browsing or using this site, you should be disappointed. 

Here at Omniplex we strive in providing our users a safe, curtious, drama free community and only ask that you follow a few simple rules. 

<span class="font-semibold">You have clearly done something to violate them or piss us off.</span>

Our Staff may approve or deny your ban appeal based on your actions and reason for ban and how much it may or may not have an effect on our community. 

We do not guarantee that your ban appeal will be approved and your ban be lifted. If you feel you have been banned for an unjust cause please Contact Us.

You can only have up to one ban appeal at any given point of time. Abusing the system will simply mean that you will not be unbanned and your ban appeal will be kept in queue.
		$v$, ARRAY[$v$Ban Appeal$v$]::text[], $v$[{"id":"reason","question":"Why were you banned?","paragraph":"Why were you banned? If you do not know, say so here and we will try to reach out.","placeholder":"I was banned because...","short":false},{"id":"why","question":"Why do you feel you should be unbanned?","paragraph":"Why do you feel you should be unbanned from the list? Have you made any changes to your conduct. Have you reflected on what you did?","placeholder":"I feel I should be unbanned because... and I have made changes to my conduct by... and I have reflected on what I did by...","short":false},{"id":"next-steps","question":"What will you do to avoid being banned in the future?","paragraph":"What will you do to avoid being banned in the future? Will you apologize if required?","placeholder":"I will avoid being banned in the future by... and I...","short":false}]$v$::jsonb, true, false, false, 0, $v$$v$, $v$banappeal$v$, 4);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$certification$v$, $v$Certification$v$, $v$So you want to apply for certification, eh?

Firstly, read the <a class="text-themable-400" href="/help/programs/certification" target="_blank">rules</a> to see if you qualify. Certification looks at whichever of servers, unique clicks, or votes your bot is strongest in you don't need to clear every bar, just one.

Then fill out the form below and wait for a staff member to review your application. If you are accepted, you're in the club!

You can only have one certification application at a time. If you have already applied, please wait for a staff member to review your application before applying again.
		$v$, ARRAY[$v$Certification$v$]::text[], $v$[{"id":"id","question":"Bot ID?","paragraph":"What is the bot ID?","placeholder":"Bot ID","short":true},{"id":"why","question":"Why do you want to be certified?","paragraph":"Why do you want to be certified? What special functionality does your bot provide that others dont?","placeholder":"I want to be certified because...","short":false},{"id":"what","question":"What does your bot do?","paragraph":"What does your bot do? Does it have NSFW functionality?","placeholder":"My bot does...","short":false}]$v$::jsonb, false, false, true, 0, $v$$v$, $v$certification$v$, 5);
INSERT INTO app_positions (id, name, info, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, behaviour, sort_order) VALUES ($v$server_certification$v$, $v$Server Certification$v$, $v$Certification isn't just for bots servers can qualify too.

Firstly, read the <a class="text-themable-400" href="/help/programs/certification" target="_blank">rules</a> to see if you qualify. Certification looks at whichever of members, unique clicks, or votes your server is strongest in you don't need to clear every bar, just one.

Then fill out the form below and wait for a staff member to review your application. If you are accepted, you're in the club!

You can only have one certification application at a time. If you have already applied, please wait for a staff member to review your application before applying again.
		$v$, ARRAY[$v$Certification$v$]::text[], $v$[{"id":"id","question":"Server ID?","paragraph":"What is the server ID?","placeholder":"Server ID","short":true},{"id":"why","question":"Why do you want your server to be certified?","paragraph":"Why do you want to be certified? What makes your community stand out from others?","placeholder":"I want to be certified because...","short":false},{"id":"what","question":"What is your server about?","paragraph":"What is your server about? Does it have NSFW content?","placeholder":"My server is about...","short":false}]$v$::jsonb, false, false, true, 0, $v$$v$, $v$server_certification$v$, 6);
UPDATE app_positions SET help_url = '/kb/programs/applications', help_label = 'How applying works' WHERE id IN ('staff', 'dev');
UPDATE app_positions SET help_url = '/kb/programs/certification', help_label = 'Certification requirements' WHERE id IN ('certification', 'server_certification');
UPDATE app_positions SET help_url = '/kb/programs/partnership', help_label = 'Partner requirements' WHERE id = 'partners';
UPDATE app_positions SET help_url = '/kb/bots/rules', help_label = 'Bot Rules' WHERE id = 'resubmit';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE app_positions;
-- +goose StatementEnd
