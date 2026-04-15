-- Portfolio Seed Script: Historical Transactions
-- This script populates a portfolio with realistic buy/sell/dividend transactions
-- spanning 18 months (Oct 2024 – Apr 2026) across 8 Finnhub-supported Common Stock tickers.
--
-- NOTE: Uses only symbols verified to be accessible on Finnhub free tier.
-- SPY and VTI (ETFs) have been replaced with TSLA and META (Common Stock).
--
-- Usage:
--   psql "$DATABASE_URL" -f backend/testdata/seed_portfolio_transactions.sql
--   OR paste contents into Supabase SQL editor
--
-- User ID: 99e69fd3-c724-496b-bd92-2386c5eb404e
-- Portfolio ID: 1b0c532b-38ba-4ffe-aa2d-c302200d5cf5

-- Apple (AAPL) - $150-190 range in Oct 2024, $231+ by Apr 2026
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AAPL', '2024-10-15', 10.0, 157.25, NULL, 1572.50, 'Initial position', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AAPL', '2024-12-10', 8.0, 172.50, NULL, 1380.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'AAPL', '2025-02-14', 18.0, NULL, 0.24, 4.32, 'Q1 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AAPL', '2025-03-20', 6.0, 181.75, NULL, 1090.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'AAPL', '2025-05-16', 24.0, NULL, 0.24, 5.76, 'Q2 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'AAPL', '2025-07-10', 5.0, 199.50, NULL, 997.50, 'Partial take profit', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'AAPL', '2025-08-15', 19.0, NULL, 0.24, 4.56, 'Q3 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AAPL', '2025-10-15', 7.0, 207.25, NULL, 1450.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'AAPL', '2025-11-15', 26.0, NULL, 0.25, 6.50, 'Q4 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AAPL', '2026-02-12', 5.0, 231.50, NULL, 1157.50, 'Q1 2026 DCA', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'AAPL', '2026-02-27', 31.0, NULL, 0.25, 7.75, 'Q1 dividend', now(), now())
ON CONFLICT DO NOTHING;

-- Microsoft (MSFT) - $370-430 range in Oct 2024, $440+ by Apr 2026
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'MSFT', '2024-11-05', 5.0, 382.50, NULL, 1912.50, 'Initial position', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'MSFT', '2025-01-15', 4.0, 398.75, NULL, 1595.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'MSFT', '2025-03-20', 9.0, NULL, 0.68, 6.12, 'Q1 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'MSFT', '2025-04-10', 3.0, 412.25, NULL, 1236.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'MSFT', '2025-06-20', 12.0, NULL, 0.68, 8.16, 'Q2 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'MSFT', '2025-08-25', 2.0, 428.50, NULL, 857.00, 'Partial exit', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'MSFT', '2025-09-18', 10.0, NULL, 0.68, 6.80, 'Q3 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'MSFT', '2025-11-12', 4.0, 435.75, NULL, 1743.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'MSFT', '2025-12-18', 14.0, NULL, 0.68, 9.52, 'Q4 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'MSFT', '2026-03-10', 3.0, 442.00, NULL, 1326.00, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;

-- Alphabet (GOOGL) - $140-160 range in Oct 2024, $180+ by Apr 2026
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'GOOGL', '2024-10-22', 8.0, 148.50, NULL, 1188.00, 'Initial position', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'GOOGL', '2025-01-08', 5.0, 158.75, NULL, 793.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'GOOGL', '2025-03-15', 6.0, 162.25, NULL, 973.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'GOOGL', '2025-06-30', 4.0, 175.50, NULL, 702.00, 'Partial take profit', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'GOOGL', '2025-09-18', 5.0, 172.00, NULL, 860.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'GOOGL', '2026-01-20', 4.0, 182.50, NULL, 730.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'GOOGL', '2026-03-25', 5.0, 187.75, NULL, 938.75, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;

-- Amazon (AMZN) - $185-205 range in Oct 2024, $225+ by Apr 2026
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AMZN', '2024-11-02', 6.0, 193.25, NULL, 1159.50, 'Initial position', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AMZN', '2025-02-10', 4.0, 205.50, NULL, 822.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AMZN', '2025-04-20', 5.0, 208.75, NULL, 1043.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'AMZN', '2025-07-15', 3.0, 218.00, NULL, 654.00, 'Partial exit', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AMZN', '2025-10-08', 4.0, 215.25, NULL, 861.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AMZN', '2026-02-01', 5.0, 226.50, NULL, 1132.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'AMZN', '2026-03-28', 3.0, 231.75, NULL, 695.25, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;

-- NVIDIA (NVDA) - $130-145 range in Oct 2024, $160+ by Apr 2026
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2024-10-08', 12.0, 135.50, NULL, 1626.00, 'Initial position', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2024-12-15', 8.0, 142.75, NULL, 1142.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2025-02-20', 7.0, 148.25, NULL, 1037.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'NVDA', '2025-05-30', 10.0, 155.50, NULL, 1555.00, 'Take profits on rally', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2025-08-12', 8.0, 151.75, NULL, 1214.00, 'DCA add after dip', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2025-11-05', 7.0, 158.25, NULL, 1107.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2026-02-18', 6.0, 162.50, NULL, 975.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'NVDA', '2026-04-05', 5.0, 167.75, NULL, 838.75, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;

-- JP Morgan (JPM) - $175-195 range in Oct 2024, $210+ by Apr 2026
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'JPM', '2024-10-18', 7.0, 184.50, NULL, 1291.50, 'Initial position', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'JPM', '2025-01-22', 5.0, 191.25, NULL, 956.25, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'JPM', '2025-03-31', 12.0, NULL, 1.10, 13.20, 'Q1 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'JPM', '2025-04-15', 4.0, 198.50, NULL, 794.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'JPM', '2025-06-30', 16.0, NULL, 1.10, 17.60, 'Q2 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'JPM', '2025-07-20', 3.0, 207.25, NULL, 621.75, 'Partial exit', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'JPM', '2025-09-30', 13.0, NULL, 1.10, 14.30, 'Q3 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'JPM', '2025-11-10', 5.0, 213.50, NULL, 1067.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'dividend', 'JPM', '2025-12-31', 18.0, NULL, 1.10, 19.80, 'Q4 dividend', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'JPM', '2026-03-05', 4.0, 220.75, NULL, 883.00, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;

-- Meta Platforms (META) - $560-740 range Oct 2024–Apr 2026 (replaces VTI ETF)
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2024-10-10', 7.0, 580.25, NULL, 4061.75, 'Initial position - AI and ads', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2024-12-15', 5.0, 610.50, NULL, 3052.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2025-02-20', 5.0, 645.75, NULL, 3228.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2025-04-15', 4.0, 668.50, NULL, 2674.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'META', '2025-06-30', 3.0, 695.25, NULL, 2085.75, 'Take profits', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2025-08-20', 5.0, 710.50, NULL, 3552.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2025-11-10', 4.0, 728.75, NULL, 2915.00, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'META', '2026-02-15', 4.0, 738.50, NULL, 2954.00, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;

-- Tesla (TSLA) - $220-360 range Oct 2024–Apr 2026 (replaces SPY ETF)
INSERT INTO public.transactions (portfolio_id, transaction_type, symbol, transaction_date, quantity, price_per_share, dividend_per_share, total_amount, notes, created_at, updated_at)
VALUES
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2024-11-08', 8.0, 240.50, NULL, 1924.00, 'Initial position - growth play', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2024-12-20', 6.0, 258.75, NULL, 1552.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2025-02-14', 5.0, 275.25, NULL, 1376.25, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'sell', 'TSLA', '2025-04-28', 4.0, 298.50, NULL, 1194.00, 'Take profits on rally', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2025-06-20', 5.0, 310.75, NULL, 1553.75, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2025-09-15', 6.0, 330.25, NULL, 1981.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2025-11-25', 5.0, 345.50, NULL, 1727.50, 'DCA add', now(), now()),
  ('1b0c532b-38ba-4ffe-aa2d-c302200d5cf5', 'buy', 'TSLA', '2026-02-24', 5.0, 358.75, NULL, 1793.75, 'Q1 2026 DCA', now(), now())
ON CONFLICT DO NOTHING;
