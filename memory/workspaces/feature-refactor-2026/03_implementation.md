# 03_implementation.md - Tech Specs & Details

## 1. Graceful Shutdown Config

### Node.js (moleculer.config.ts)
```typescript
tracking: {
    enabled: true,
    shutdownTimeout: 30000 // 30s default
}
```

### Go Service (main.go)
- Capture `SIGTERM`.
- Use `server.Shutdown(ctx)`.
- **Critical**: Separate DB Context from HTTP Context.

### Kubernetes (deployment.yaml)
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 10"]
terminationGracePeriodSeconds: 45 # (or 75 for Connectors)
```

## 2. Retry Policy (Moleculer)
```javascript
retryPolicy: {
    enabled: true,
    retries: 3,
    delay: 500,
    check: (err) => err && err.retryable
}
```

## 3. Idempotency Strategy
- **DB**: Add `UNIQUE INDEX` on `request_id` or `reference_code`.
- **Redis**: Use distributed lock with TTL.
