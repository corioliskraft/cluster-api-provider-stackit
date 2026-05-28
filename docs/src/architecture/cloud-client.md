# Cloud Client

The cloud client abstraction hides STACKIT SDK details from controllers.

It covers:

- Server create, get, list, and delete
- Network lookup
- Load balancer create, list, delete, and target-pool updates
- Provider ID helpers
- Error classification

Controllers should handle classified errors differently:

- transient errors should be retried
- terminal validation errors should set clear conditions
- not-found errors during deletion should be treated as successful cleanup where
  appropriate

Tests use an in-memory fake cloud client for deterministic reconciliation
coverage. SDK integration tests are opt-in and require real STACKIT credentials.
