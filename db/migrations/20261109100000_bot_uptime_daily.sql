-- +goose Up
-- +goose StatementBegin
CREATE TABLE bot_uptime_daily (
    bot_id text NOT NULL,
    day date NOT NULL,
    checks integer DEFAULT 0 NOT NULL,
    online integer DEFAULT 0 NOT NULL,
    CONSTRAINT bot_uptime_daily_pkey PRIMARY KEY (bot_id, day),
    CONSTRAINT bot_uptime_daily_bot_fkey FOREIGN KEY (bot_id) REFERENCES bots(bot_id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX bot_uptime_daily_day_idx ON bot_uptime_daily (day);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE bot_uptime_daily;
-- +goose StatementEnd
