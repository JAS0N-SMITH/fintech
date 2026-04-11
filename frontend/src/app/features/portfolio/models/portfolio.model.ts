/** Represents a named brokerage account grouping owned by a user. */
export interface Portfolio {
  id: string;
  user_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

/** Input for creating a new portfolio. */
export interface CreatePortfolioInput {
  name: string;
  description?: string;
}

/** Input for updating an existing portfolio. */
export interface UpdatePortfolioInput {
  name: string;
  description?: string;
}
