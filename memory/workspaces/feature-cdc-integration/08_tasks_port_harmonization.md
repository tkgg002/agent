# Task: CDC Port Harmonization Execution

Dọn dẹp tiến trình treo và đồng bộ hóa bộ cổng dịch vụ 8081-8082-8083.

- `[x]` 1. Cleanup Ghost Processes:
    - `[x]` Kill PID 31852 (on 8081)
    - `[x]` Kill PID 70239 (on 8082)
    - `[x]` Kill PID 69840 (on 8090)
- `[x]` 2. Backend Harmonization (Auth/CMS/Worker):
    - `[x]` Update `cdc-auth-service/config/config-local.yml` (Port: 8081)
    - `[x]` Update `centralized-data-service/config/config-local.yml` (Port: 8082)
    - `[x]` Update `cdc-cms-service/config/config-local.yml` (Port: 8083)
- `[x]` 3. Frontend Internalization:
    - `[x]` Update `cdc-cms-web/.env` (8081, 8082, 8083)
- `[x]` 4. Verification:
    - `[x]` Run `lsof` check
    - `[x]` Test connectivity
- `[x]` 5. Governance:
    - `[x]` Update `05_progress.md`
