-- database/migrations/000001_trade_schema_v2.down.sql
BEGIN;

-- Drop columns we added
ALTER TABLE trades
DROP COLUMN IF EXISTS exchange,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS sentiment,
    DROP COLUMN IF EXISTS indicators,
    DROP COLUMN IF EXISTS fee,
    DROP COLUMN IF EXISTS exit_price,
    DROP COLUMN IF EXISTS entry_price,
    DROP COLUMN IF EXISTS is_long,
    DROP COLUMN IF EXISTS is_short,
    DROP COLUMN IF EXISTS take_profit,
    DROP COLUMN IF EXISTS stop_loss,
    DROP COLUMN IF EXISTS is_reduce_only,
    DROP COLUMN IF EXISTS is_take_profit,
    DROP COLUMN IF EXISTS stop_price,
    DROP COLUMN IF EXISTS price,
    DROP COLUMN IF EXISTS order_type,
    DROP COLUMN IF EXISTS asset_mode,
    DROP COLUMN IF EXISTS leverage,
    DROP COLUMN IF EXISTS margin_mode;

-- Change quantity back to int (precision loss)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'trades' AND column_name = 'quantity'
    ) THEN
        EXECUTE 'ALTER TABLE trades ALTER COLUMN quantity TYPE integer USING round(quantity)';
END IF;
END $$;

-- Rename is_short -> is_shot if needed
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'trades' AND column_name = 'is_short'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'trades' AND column_name = 'is_shot'
    )
    THEN
        EXECUTE 'ALTER TABLE trades RENAME COLUMN is_short TO is_shot';
END IF;
END $$;

COMMIT;
