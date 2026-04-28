# Plan — Phase 15 Retry Failed Scope Enrichment

## Kế hoạch thực hiện

1. Audit retry endpoint ở CMS backend.
2. Audit downstream consumer `cdc.cmd.retry-failed` ở worker để xác định backward compatibility.
3. Chốt quyết định kiến trúc:
   - giữ ID là canonical identity
   - enrich response + downstream payload bằng source/shadow scope
4. Refactor FE `DataIntegrity` để dùng metadata failed-log trực tiếp.
5. Verify backend tests + FE build.
