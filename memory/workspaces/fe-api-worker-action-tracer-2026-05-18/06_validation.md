# Validation Plan

- FE: run available build/test/lint command if package scripts exist.
- API: run available build/test/lint command if package scripts exist.
- Worker: run focused Go tests/build for touched packages.
- Runtime checks:
  - verify FE/API/worker service health where ports are discoverable.
  - verify command path by source-level trace and, if safe, by local endpoint invocation.
- Security scan changed files for secrets and unsafe user-input SQL/publish paths.

