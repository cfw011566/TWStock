-- DROP TABLE day_trade_securities;

CREATE TABLE day_trade_securities (
	trade_date      date,    -- trade date
	security_code   varchar,
	volume			numeric,
	buy_value		numeric,
	sell_value		numeric,
	UNIQUE (trade_date, security_code)
);

-- DROP TABLE day_trade_indices;

CREATE TABLE day_trade_indices (
	trade_date      date,    -- trade date
	security_code   varchar,
	volume			numeric,
	volume_percent	numeric,
	buy_value		numeric,
	buy_percent		numeric,
	sell_value		numeric,
	sell_percent	numeric,
	UNIQUE (trade_date, security_code)
);
