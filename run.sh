#!/bin/sh

exec bingsoo \
  --port "$PORT" \
  --concurrency "$CONCURRENCY" \
  --signing-secret "$SIGNING_SECRET" \
  --postgres-host "$POSTGRES_HOST" \
  --postgres-user "$POSTGRES_USER" \
  --postgres-password "$POSTGRES_PASSWORD" \
  --postgres-db "$POSTGRES_DB" \
  --redis-host "$REDIS_HOST"
