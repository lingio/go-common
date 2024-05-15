# spanner

Spanner utilities:

- `ChangeStreamReader` with `ChangeStreamHandler`
- `DetectDialect` for postgres / google sql dialect

# integration testing

Using default credentials:

```
SPANNER_TEST_PROJECT_ID=lingio-stage \
SPANNER_TEST_INSTANCE_ID=lingio-staging-1 \
SPANNER_TEST_DATABASE_ID=test \
go test ./spanner
```

otherwise, add `SPANNER_TEST_CREDENTIALS_BASE64=base64-encoded-jsoncreds`.
