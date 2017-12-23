
CREATE TABLE indices_5seconds (
	trade_datetime	timestamp,
	security_code	varchar,
	index_value		numeric,
	UNIQUE (trade_datetime, security_code)
);

CREATE TABLE trades_5seconds (
	trade_datetime	timestamp,
	security_code	varchar,
	acc_bid_order	numeric,
	acc_bid_volume	numeric,
	acc_ask_order	numeric,
	acc_ask_volume	numeric,
	acc_trade_count	numeric,
	acc_trade_volume	numeric,
	acc_trade_amount	numeric,
	bid_order		numeric,
	bid_volume		numeric,
	ask_order		numeric,
	ask_volume		numeric,
	trade_count		numeric,
	trade_volume	numeric,
	trade_amount	numeric,
	UNIQUE (trade_datetime, security_code)
);
