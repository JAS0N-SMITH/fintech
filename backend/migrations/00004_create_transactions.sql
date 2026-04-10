-- +goose Up
-- +goose StatementBegin

-- transactions is the single source of truth for all financial events.
-- Current holdings, cost basis, and gain/loss are always derived — never stored.
CREATE TABLE public.transactions (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id         uuid        NOT NULL REFERENCES public.portfolios(id) ON DELETE CASCADE,
    transaction_type     text        NOT NULL CHECK (
                                         transaction_type IN ('buy', 'sell', 'dividend', 'reinvested_dividend')
                                     ),
    symbol               text        NOT NULL CHECK (
                                         symbol ~ '^[A-Z0-9.\-]{1,20}$'
                                     ),
    transaction_date     date        NOT NULL,
    -- quantity is NULL for pure cash dividends (dividend_per_share only).
    quantity             numeric(18,8) CHECK (quantity > 0),
    -- price_per_share is NULL for cash dividends.
    price_per_share      numeric(18,8) CHECK (price_per_share >= 0),
    -- dividend_per_share is only set for dividend and reinvested_dividend types.
    dividend_per_share   numeric(18,8) CHECK (dividend_per_share > 0),
    total_amount         numeric(18,8) NOT NULL CHECK (total_amount > 0),
    notes                text         CHECK (char_length(notes) <= 1000),
    created_at           timestamptz  NOT NULL DEFAULT now(),
    updated_at           timestamptz  NOT NULL DEFAULT now(),

    -- Enforce field presence per transaction type.
    CONSTRAINT transactions_buy_sell_fields CHECK (
        transaction_type NOT IN ('buy', 'sell', 'reinvested_dividend')
        OR (quantity IS NOT NULL AND price_per_share IS NOT NULL)
    ),
    CONSTRAINT transactions_dividend_fields CHECK (
        transaction_type NOT IN ('dividend', 'reinvested_dividend')
        OR dividend_per_share IS NOT NULL
    )
);

CREATE INDEX transactions_portfolio_id_idx ON public.transactions(portfolio_id);
CREATE INDEX transactions_symbol_idx       ON public.transactions(symbol);
CREATE INDEX transactions_date_idx         ON public.transactions(transaction_date DESC);

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON public.transactions
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.transactions ENABLE ROW LEVEL SECURITY;

-- Users can only access transactions belonging to their own portfolios.
CREATE POLICY "transactions_select_own"
    ON public.transactions FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM public.portfolios p
            WHERE p.id = portfolio_id AND p.user_id = auth.uid()
        )
    );

CREATE POLICY "transactions_insert_own"
    ON public.transactions FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM public.portfolios p
            WHERE p.id = portfolio_id AND p.user_id = auth.uid()
        )
    );

CREATE POLICY "transactions_update_own"
    ON public.transactions FOR UPDATE
    USING (
        EXISTS (
            SELECT 1 FROM public.portfolios p
            WHERE p.id = portfolio_id AND p.user_id = auth.uid()
        )
    );

CREATE POLICY "transactions_delete_own"
    ON public.transactions FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM public.portfolios p
            WHERE p.id = portfolio_id AND p.user_id = auth.uid()
        )
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.transactions;

-- +goose StatementEnd
