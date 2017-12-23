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

CREATE TABLE day_trade_securities (
	trade_date      date,    -- trade date
	security_code   varchar,
	volume			numeric,
	buy_value		numeric,
	sell_value		numeric,
	UNIQUE (trade_date, security_code)
);

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

ALTER TABLE daily_investors
    ADD foreign_self_buy    numeric,
    ADD foreign_self_sell   numeric,
    ADD foreign_self_diff   numeric;


-- TAIEX & Group Indices per 5 Seconds

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


-- Margin Transaction 融資Purchase on Margin/融券Short Sale

CREATE TABLE daily_margin_short (
	trade_date		date,
	security_code	varchar,
	margin_new			numeric,	-- 融資買進
	margin_redemption	numeric,	-- 融資賣出
	margin_outstanding	numeric,	-- 現金償還
	margin_last_remain	numeric,	-- 前日餘額
	margin_remain		numeric,	-- 今日餘額
	margin_limit		numeric,	-- 限額
	short_redemption	numeric,	-- 融券買進
	short_new			numeric,	-- 融券賣出
	short_outstanding	numeric,
	short_last_remain	numeric,
	short_remain		numeric,
	short_limit			numeric,
	margin_and_short	numeric,	-- 資券互抵
	UNIQUE (trade_date, security_code)
);


-- Indices information (TAIEX/OTC...)

CREATE TABLE index_values (
	trade_date		date,
	index_code		varchar,	-- TAIEX / OTC
	close_value     numeric,
	open_value      numeric,
	highest_value   numeric,
	lowest_value    numeric,
	trade_volume	numeric,  -- shares
	trade_amount    numeric,
	trade_count     numeric,  -- transcation
	UNIQUE (trade_date, index_code)
);

CREATE TABLE index_investors (
	trade_date			date,
	index_code			varchar,	-- TAIEX / OTC
	dealer_self_buy		numeric,  -- dealer (proprietary)
	dealer_self_sell	numeric,  -- dealer (proprietary)
	dealer_self_diff	numeric,  -- dealer (proprietary)
	dealer_hedge_buy	numeric,
	dealer_hedge_sell	numeric,
	dealer_hedge_diff	numeric,
	trust_buy			numeric,
	trust_sell			numeric,
	trust_diff			numeric,
	foreign_buy			numeric,
	foreign_sell		numeric,
	foreign_diff		numeric,
	total_buy			numeric,
	total_sell			numeric,
	total_diff			numeric,
	UNIQUE (trade_date, index_code)
);

ALTER TABLE index_investors
	ADD foreign_self_buy	numeric,
	ADD foreign_self_sell	numeric,
	ADD foreign_self_diff	numeric;

CREATE TABLE index_margin_short (
	trade_date			date,
	index_code			varchar,	-- TAIEX / OTC
	margin_new			numeric,	-- 融資買進
	margin_redemption	numeric,	-- 融資賣出
	margin_outstanding	numeric,	-- 現金償還
	margin_last_remain	numeric,	-- 前日餘額
	margin_remain		numeric,	-- 今日餘額
	short_redemption	numeric,	-- 融券買進
	short_new			numeric,	-- 融券賣出
	short_outstanding	numeric,
	short_last_remain	numeric,
	short_remain		numeric,
	margin_new_value			numeric,	-- 融資買進
	margin_redemption_value		numeric,	-- 融資賣出
	margin_outstanding_value	numeric,	-- 現金償還
	margin_last_remain_value	numeric,	-- 前日餘額
	margin_remain_value			numeric,	-- 今日餘額
	UNIQUE (trade_date, index_code)
);

