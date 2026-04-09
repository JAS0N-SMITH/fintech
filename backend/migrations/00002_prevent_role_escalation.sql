-- +goose Up
-- +goose StatementBegin

-- Prevent users from modifying their own role via the UPDATE policy on profiles.
-- The profiles_update_own RLS policy allows self-updates but doesn't restrict
-- which columns can change. This trigger enforces that role is immutable via
-- the Data API — only a service-role or admin operation can change it.
CREATE OR REPLACE FUNCTION public.prevent_role_escalation()
RETURNS TRIGGER LANGUAGE plpgsql
SET search_path = '' AS $$
BEGIN
    IF NEW.role IS DISTINCT FROM OLD.role THEN
        RAISE EXCEPTION 'role modification is not permitted';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER profiles_prevent_role_escalation
    BEFORE UPDATE ON public.profiles
    FOR EACH ROW EXECUTE FUNCTION public.prevent_role_escalation();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS profiles_prevent_role_escalation ON public.profiles;
DROP FUNCTION IF EXISTS public.prevent_role_escalation();

-- +goose StatementEnd
