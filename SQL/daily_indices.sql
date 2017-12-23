CREATE TABLE daily_indices (
	trade_date      date,    -- trade date
	security_code   varchar,
	index_value		numeric,
	trade_volume	numeric,  -- shares
	trade_amount    numeric,
	trade_count     numeric,  -- transcation
	UNIQUE (trade_date, security_code)
);
