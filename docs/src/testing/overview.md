# Testing Overview

Testing is split into layers:

- unit tests for helper functions and cloud logic
- envtest controller tests with Kubernetes API machinery
- opt-in SDK integration tests against a real STACKIT project
- opt-in e2e tests that create billable STACKIT resources

Default test command:

```sh
make test
```

Real cloud tests are gated by environment variables so they do not run
accidentally.
