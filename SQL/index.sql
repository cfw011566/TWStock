
-- Indices information (TAIEX/OTC...)

DROP TABLE index_values;

CREATE TABLE index_values (
	trade_date		date,
	index_code		varchar,	-- TAIEX / OTC
	open_value      numeric,
	highest_value   numeric,
	lowest_value    numeric,
	close_value     numeric,
	trade_volume	numeric,  -- shares
	trade_amount    numeric,
	trade_count     numeric,  -- transcation
	UNIQUE (trade_date, index_code)
);

DROP TABLE index_investors;

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
    ADD foreign_self_buy    numeric,
    ADD foreign_self_sell   numeric,
    ADD foreign_self_diff   numeric;

DROP TABLE index_margin_short;

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
