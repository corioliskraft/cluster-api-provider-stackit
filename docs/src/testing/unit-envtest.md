# Unit and Envtest

Run the non-e2e suite:

```sh
make test
```

This runs:

- generated manifest checks through controller-gen
- `go fmt`
- `go vet`
- envtest-backed Go tests, excluding `/e2e`

Coverage includes:

- provider ID helpers
- credential parsing
- cloud error classification
- fake cloud client behavior
- cluster and machine reconciliation
- load balancer target registration and cleanup
- paused handling and related-object watches
