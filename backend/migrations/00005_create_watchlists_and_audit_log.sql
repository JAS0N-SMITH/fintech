-- +goose Up
-- +goose StatementBegin

-- watchlists is the header record for a named list of tracked tickers.
CREATE TABLE public.watchlists (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name       text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX watchlists_user_id_idx ON public.watchlists(user_id);

CREATE TRIGGER watchlists_set_updated_at
    BEFORE UPDATE ON public.watchlists
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.watchlists ENABLE ROW LEVEL SECURITY;

CREATE POLICY "watchlists_select_own" ON public.watchlists FOR SELECT USING (auth.uid() = user_id);
CREATE POLICY "watchlists_insert_own" ON public.watchlists FOR INSERT WITH CHECK (auth.uid() = user_id);
CREATE POLICY "watchlists_update_own" ON public.watchlists FOR UPDATE USING (auth.uid() = user_id) WITH CHECK (auth.uid() = user_id);
CREATE POLICY "watchlists_delete_own" ON public.watchlists FOR DELETE USING (auth.uid() = user_id);

-- watchlist_items is an individual ticker entry on a watchlist.
-- Current price comes from live market data, never stored here.
CREATE TABLE public.watchlist_items (
    id           uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    watchlist_id uuid          NOT NULL REFERENCES public.watchlists(id) ON DELETE CASCADE,
    symbol       text          NOT NULL CHECK (symbol ~ '^[A-Z0-9.\-]{1,20}$'),
    target_price numeric(18,8) CHECK (target_price > 0),
    notes        text          CHECK (char_length(notes) <= 500),
    created_at   timestamptz   NOT NULL DEFAULT now(),
    updated_at   timestamptz   NOT NULL DEFAULT now(),

    UNIQUE (watchlist_id, symbol)
);

CREATE INDEX watchlist_items_watchlist_id_idx ON public.watchlist_items(watchlist_id);

CREATE TRIGGER watchlist_items_set_updated_at
    BEFORE UPDATE ON public.watchlist_items
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE public.watchlist_items ENABLE ROW LEVEL SECURITY;

-- Items inherit access from their parent watchlist via subquery.
CREATE POLICY "watchlist_items_select_own"
    ON public.watchlist_items FOR SELECT
    USING (EXISTS (SELECT 1 FROM public.watchlists w WHERE w.id = watchlist_id AND w.user_id = auth.uid()));

CREATE POLICY "watchlist_items_insert_own"
    ON public.watchlist_items FOR INSERT
    WITH CHECK (EXISTS (SELECT 1 FROM public.watchlists w WHERE w.id = watchlist_id AND w.user_id = auth.uid()));

CREATE POLICY "watchlist_items_update_own"
    ON public.watchlist_items FOR UPDATE
    USING (EXISTS (SELECT 1 FROM public.watchlists w WHERE w.id = watchlist_id AND w.user_id = auth.uid()));

CREATE POLICY "watchlist_items_delete_own"
    ON public.watchlist_items FOR DELETE
    USING (EXISTS (SELECT 1 FROM public.watchlists w WHERE w.id = watchlist_id AND w.user_id = auth.uid()));

-- audit_log is append-only. Never UPDATE or DELETE rows.
-- Stores security-relevant events: login, role changes, transactions, admin actions.
-- PII is masked before storage.
CREATE TABLE public.audit_log (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid        REFERENCES auth.users(id) ON DELETE SET NULL,
    action        text        NOT NULL,
    target_entity text,
    target_id     uuid,
    before_value  jsonb,
    after_value   jsonb,
    ip_address    inet,
    user_agent    text        CHECK (char_length(user_agent) <= 500),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_user_id_idx    ON public.audit_log(user_id);
CREATE INDEX audit_log_action_idx     ON public.audit_log(action);
CREATE INDEX audit_log_created_at_idx ON public.audit_log(created_at DESC);

ALTER TABLE public.audit_log ENABLE ROW LEVEL SECURITY;

-- Users can read their own audit entries.
CREATE POLICY "audit_log_select_own"
    ON public.audit_log FOR SELECT
    USING (auth.uid() = user_id);

-- Admins can read all audit entries.
CREATE POLICY "audit_log_select_admin"
    ON public.audit_log FOR SELECT
    USING (EXISTS (SELECT 1 FROM public.profiles p WHERE p.id = auth.uid() AND p.role = 'admin'));

-- No UPDATE or DELETE policies — audit_log is append-only.
-- INSERT is handled by service-role / server-side code only (no user-facing policy).

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS public.watchlist_items;
DROP TABLE IF EXISTS public.watchlists;
DROP TABLE IF EXISTS public.audit_log;

-- +goose StatementEnd
