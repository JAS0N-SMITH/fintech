-- +goose Up
-- +goose StatementBegin

-- portfolios represents a named brokerage account grouping.
-- No financial values are stored here — all are derived from transactions.
CREATE TABLE public.portfolios (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name        text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    description text CHECK (char_length(description) <= 500),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX portfolios_user_id_idx ON public.portfolios(user_id);

CREATE TRIGGER portfolios_set_updated_at
    BEFORE UPDATE ON public.portfolios
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.portfolios ENABLE ROW LEVEL SECURITY;

-- Users can only see their own portfolios.
CREATE POLICY "portfolios_select_own"
    ON public.portfolios FOR SELECT
    USING (auth.uid() = user_id);

-- Users can create portfolios for themselves only.
CREATE POLICY "portfolios_insert_own"
    ON public.portfolios FOR INSERT
    WITH CHECK (auth.uid() = user_id);

-- Users can update their own portfolios.
CREATE POLICY "portfolios_update_own"
    ON public.portfolios FOR UPDATE
    USING (auth.uid() = user_id)
    WITH CHECK (auth.uid() = user_id);

-- Users can delete their own portfolios.
CREATE POLICY "portfolios_delete_own"
    ON public.portfolios FOR DELETE
    USING (auth.uid() = user_id);

-- Admins can read all portfolios.
CREATE POLICY "portfolios_select_admin"
    ON public.portfolios FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM public.profiles p
            WHERE p.id = auth.uid() AND p.role = 'admin'
        )
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.portfolios;

-- +goose StatementEnd
