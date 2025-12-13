-- database/migrations/000001_trade_schema_v2.up.sql
BEGIN;

-- 1) Rename is_shot -> is_short (only if the old name exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'trades' AND column_name = 'is_shot'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'trades' AND column_name = 'is_short'
    )
    THEN
        EXECUTE 'ALTER TABLE trades RENAME COLUMN is_shot TO is_short';
END IF;
END $$;

-- 2) Change quantity from int -> double precision (float8)
--    USING clause preserves/casts existing values.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'trades' AND column_name = 'quantity'
    ) THEN
        EXECUTE 'ALTER TABLE trades ALTER COLUMN quantity TYPE double precision USING quantity::double precision';
END IF;
END $$;

-- 3) Add new columns (mostly nullable first, booleans NOT NULL with default)
ALTER TABLE trades
    ADD COLUMN IF NOT EXISTS margin_mode  text,
    ADD COLUMN IF NOT EXISTS leverage     double precision,
    ADD COLUMN IF NOT EXISTS asset_mode   text,
    ADD COLUMN IF NOT EXISTS order_type   text,
    ADD COLUMN IF NOT EXISTS price        double precision,
    ADD COLUMN IF NOT EXISTS stop_price   double precision,
    ADD COLUMN IF NOT EXISTS is_take_profit boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_reduce_only boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS stop_loss    double precision,
    ADD COLUMN IF NOT EXISTS take_profit  double precision,
    ADD COLUMN IF NOT EXISTS is_short     boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_long      boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS entry_price  double precision,
    ADD COLUMN IF NOT EXISTS exit_price   double precision,
    ADD COLUMN IF NOT EXISTS fee          double precision,
    ADD COLUMN IF NOT EXISTS indicators   text,
    ADD COLUMN IF NOT EXISTS sentiment    text,
    ADD COLUMN IF NOT EXISTS notes        text,
    ADD COLUMN IF NOT EXISTS exchange     text;

-- 4) (Optional) trade_date type normalize to timestamptz
-- Uncomment if you want timezone-aware storage. Adjust timezone as appropriate.
-- ALTER TABLE trades
--     ALTER COLUMN trade_date TYPE timestamptz
--     USING (CASE
--              WHEN pg_typeof(trade_date)::text = 'timestamp without time zone'
--              THEN trade_date AT TIME ZONE 'UTC'
--              ELSE trade_date
--            END);

COMMIT;
