-- Margin Transaction 融資Purchase on Margin/融券Short Sale
DROP TABLE daily_margin_short;

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
