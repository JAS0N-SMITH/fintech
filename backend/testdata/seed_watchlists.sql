-- Watchlist Seed Script
-- Creates 3 themed watchlists with 19 items total
--
-- NOTE: Uses only symbols verified to be accessible on Finnhub free tier.
-- SCHD (dividend ETF) has been replaced with MCD (Common Stock).
--
-- Usage:
--   psql "$DATABASE_URL" -f backend/testdata/seed_watchlists.sql
--   OR paste contents into Supabase SQL editor
--
-- User ID: 99e69fd3-c724-496b-bd92-2386c5eb404e

-- Watchlist headers
INSERT INTO public.watchlists (id, user_id, name, created_at, updated_at)
VALUES
  ('a1b2c3d4-0001-4000-8000-000000000001', '99e69fd3-c724-496b-bd92-2386c5eb404e', 'AI & Semiconductors', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000002', '99e69fd3-c724-496b-bd92-2386c5eb404e', 'Dividend Income', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', '99e69fd3-c724-496b-bd92-2386c5eb404e', 'Potential Buys', now(), now())
ON CONFLICT DO NOTHING;

-- AI & Semiconductors (6 items)
INSERT INTO public.watchlist_items (watchlist_id, symbol, target_price, notes, created_at, updated_at)
VALUES
  ('a1b2c3d4-0001-4000-8000-000000000001', 'AMD',  175.00, 'AI GPU competitor to NVDA, watching for entry below $175', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000001', 'AVGO', 220.00, 'Broadcom - AI networking and custom chips, strong dividend', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000001', 'QCOM', 155.00, 'Qualcomm - on-device AI, watching for mobile cycle rebound', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000001', 'TSM',  175.00, 'TSMC ADR - world''s largest chip foundry, NVDA/AAPL supplier', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000001', 'ARM',  120.00, 'ARM Holdings - CPU architecture licensing, AI edge compute', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000001', 'INTC',  22.00, 'Intel - deep value turnaround play, waiting for confirmation', now(), now())
ON CONFLICT DO NOTHING;

-- Dividend Income (6 items)
INSERT INTO public.watchlist_items (watchlist_id, symbol, target_price, notes, created_at, updated_at)
VALUES
  ('a1b2c3d4-0001-4000-8000-000000000002', 'O',    52.00, 'Realty Income - monthly dividend REIT, target yield >5%', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000002', 'KO',   58.00, 'Coca-Cola - Dividend King, 60+ years of consecutive raises', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000002', 'PG',  155.00, 'Procter & Gamble - Dividend King, defensive consumer staples', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000002', 'JNJ',  145.00, 'Johnson & Johnson - Dividend King, healthcare stability', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000002', 'MCD',  295.00, 'McDonald''s - Dividend growth story, asset-light model', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000002', 'T',    18.00, 'AT&T - high yield ~6%, watching debt paydown progress', now(), now())
ON CONFLICT DO NOTHING;

-- Potential Buys (7 items)
INSERT INTO public.watchlist_items (watchlist_id, symbol, target_price, notes, created_at, updated_at)
VALUES
  ('a1b2c3d4-0001-4000-8000-000000000003', 'META',  520.00, 'Meta - AI ad targeting + Reality Labs optionality', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', 'BRK.B', 420.00, 'Berkshire Hathaway - value compounder, Buffett succession watch', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', 'V',     290.00, 'Visa - payments network moat, cashless tailwind', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', 'COST',  870.00, 'Costco - recession-resistant, loyal membership model', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', 'NFLX',  950.00, 'Netflix - ad tier growth, password sharing crackdown payoff', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', 'UNH',   480.00, 'UnitedHealth - managed care leader, watching regulatory risk', now(), now()),
  ('a1b2c3d4-0001-4000-8000-000000000003', 'LLY',   800.00, 'Eli Lilly - GLP-1 weight loss drugs, long runway for growth', now(), now())
ON CONFLICT DO NOTHING;
