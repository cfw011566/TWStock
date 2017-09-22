CREATE TABLE daily_quotes (
	trade_date      date,    -- trade date
	security_code   varchar,
	close_price     numeric,
	price_change    numeric,
	open_price      numeric,
	highest_price   numeric,
	lowest_price    numeric,
	trade_volume	numeric,  -- shares
	trade_amount    numeric,
	trade_count     numeric,  -- transcation
	last_bid_price  numeric,
	last_bid_volume numeric,
	last_ask_price  numeric,
	last_ask_volume numeric,
	CHECK (highest_price >= lowest_price),
	UNIQUE (trade_date, security_code)
);


CREATE TABLE daily_indices (
	trade_date      date,    -- trade date
	security_code   varchar,
	index_value		numeric,
	trade_volume	numeric,  -- shares
	trade_amount    numeric,
	trade_count     numeric,  -- transcation
	UNIQUE (trade_date, security_code)
);


--  Trading Volume of Foreign & Other Investors (Share)

CREATE TABLE daily_investors (
	trade_date      date,    -- trade date
	security_code   varchar,
	foreign_buy		numeric,
	foreign_sell	numeric,
	foreign_diff	numeric,
	trust_buy		numeric,
	trust_sell		numeric,
	trust_diff		numeric,
	dealer_diff		numeric,
	dealer_self_buy	numeric,  -- dealer (proprietary)
	dealer_self_sell	numeric,  -- dealer (proprietary)
	dealer_self_diff	numeric,  -- dealer (proprietary)
	dealer_hedge_buy	numeric,
	dealer_hedge_sell	numeric,
	dealer_hedge_diff	numeric,
	investors_diff	numeric,
	UNIQUE (trade_date, security_code)
);


-- TAIEX & Group Indices per 5 Seconds

CREATE TABLE indices (
	trade_datetime	timestamp,
	security_code	varchar,
	index_value		numeric,
	UNIQUE (trade_datetime, security_code)
);
