# Validation

- `cdc-cms-web`: `npm run build` PASS after removing Flow1 pages/routes and rewriting `SourceConnectors`.
- `cdc-cms-service`: `go test ./...` PASS after adding connector config update support.
- Browser: login with local admin account succeeded after starting `cdc-auth-service` on `:8081`.
- Browser: `/sources` loads and shows `New Connect`, `Connections`, `Source Fingerprints`, `Connectors`.
- Browser: `New Connect` modal verified dynamic DB-type switching for `MongoDB`, `MySQL`, `PostgreSQL`, with different config fields per type.
- Browser: `Edit Config` on linked connector `market-mongodb-cdc` successfully calls PATCH update endpoint and shows success toast `Connector config updated`.
- Browser: `/shadow` route loads with heading `Shadow`; old `flow1` menu is absent from navigation.
- Runtime verification: stale CMS binary initially returned `404 Cannot PATCH /api/v1/system/connectors/:name/config`; restarting latest service removed the route mismatch.
- Runtime verification: first PATCH returned `200` but emitted DB warning because `cdc_system.sources.status='updated'` violated `sources_status_check`; code fixed and clean PATCH re-verified without warning.
