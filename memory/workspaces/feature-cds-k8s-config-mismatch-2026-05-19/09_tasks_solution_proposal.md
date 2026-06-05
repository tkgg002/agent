# 09_tasks_solution_proposal.md — Đề xuất hướng đi (chờ user approve)

> **Brain Code Prohibition (CLAUDE.md §12)**: tài liệu này chỉ đề xuất.
> Mọi thay đổi `.go` / `.yaml` phải do Muscle thực hiện sau khi user chọn hướng.

## 3 hướng giải quyết — chọn 1 (hoặc phối hợp)

### Hướng 1 — "Sửa manifest cho khớp config.go" (rẻ, nhanh)

**Phạm vi**: chỉ sửa `deployments/k8s/cdc-worker-deployment.yaml` + Secret `cdc-secrets`.

**Việc cần làm** (Muscle sẽ thực hiện):
1. Đổi ENV name theo struct path Viper auto-derive:
   - `WORKER_POOL_SIZE` → `WORKER_POOLSIZE`
   - `BATCH_SIZE` → `WORKER_BATCHSIZE`
   - `BATCH_TIMEOUT` → `WORKER_BATCHTIMEOUT`
2. Xóa `DB_SINK_URL` (không dùng).
3. Thêm các ENV bắt buộc từ secretKeyRef:
   - `SYSTEMDB_HOST`, `SYSTEMDB_DATABASE`, `SYSTEMDB_USER`, `SYSTEMDB_PASSWORD`, `SYSTEMDB_PORT`
   - `MASTERDB_HOST`, `MASTERDB_DATABASE`, `MASTERDB_USER`, `MASTERDB_PASSWORD`, `MASTERDB_PORT`
   - `JWT_SECRET`
   - `KAFKA_BROKERS`, `KAFKA_CONNECT_URL`, `KAFKA_SCHEMA_REGISTRY_URL`
   - `MASTER_KEY`, `OTEL_ENDPOINT`, `DEBEZIUM_CONNECTOR_NAME`
4. Pin image: `cdc-worker:v1.2.3@sha256:...` thay vì `:latest`.
5. Thêm key tương ứng vào Secret `cdc-secrets`.

**Ưu**: nhanh, không đụng Go code, Pod start được ngay.
**Nhược**: vẫn để generic ENV name (`JWT_SECRET`, `MASTER_KEY`) — review security có thể vẫn comment.

---

### Hướng 2 — "Thêm prefix project cho mọi ENV" (an toàn, mid-cost)

**Phạm vi**: sửa cả `config.go` (đổi prefix Viper) + manifest.

**Việc cần làm**:
1. `config.go`: `v.SetEnvPrefix("CDS")` → Viper auto-derive `CDS_SYSTEMDB_HOST`, `CDS_JWT_SECRET`, …
2. Sửa hardcoded keys trong `applyEnvOverrides` thành `CDS_*`.
3. Manifest đổi mọi ENV sang prefix `CDS_*`.
4. Update `README.md` section "Environment variables".

**Ưu**: tránh xung đột với sidecar/platform reserved keys; security review pass dễ hơn; dễ grep audit.
**Nhược**: phải sửa Go code (Muscle scope, không phải Brain) + cập nhật toàn bộ doc + runbook + Makefile env.

---

### Hướng 3 — "Đóng gói qua Helm + ConfigMap" (đúng chuẩn, đắt nhất)

**Phạm vi**: refactor toàn bộ `deployments/` thành Helm chart.

**Việc cần làm**:
1. Tạo `deployments/helm/centralized-data-service/`:
   - `Chart.yaml`, `values.yaml` (per-env: dev/staging/prod), `templates/deployment.yaml`, `templates/configmap.yaml`, `templates/secret.yaml`, `templates/hpa.yaml`, `templates/service.yaml`.
2. ConfigMap chứa non-secret config; Secret chứa credentials.
3. Mount config qua `envFrom: configMapRef` + `envFrom: secretRef` thay vì khai từng ENV.
4. Pin image qua `values.yaml` (`image.tag`, `image.digest`).
5. Helm hooks cho migration (`pre-install` chạy `make migrate-bootstrap`).
6. Bỏ `deployments/k8s/cdc-worker-deployment.yaml` cũ.

**Ưu**: 1 nguồn truth, đa env, immutable manifest, CI/CD friendly, security gate pass dễ.
**Nhược**: tốn 1-2 ngày; team cần biết Helm; phải migrate Secret hiện tại.

---

### Hướng 4 — "Port pattern `cdc-cms-service`" ⭐ KHUYẾN NGHỊ MỚI

**Phạm vi**: refactor `centralized-data-service/config/config.go` theo y chang khuôn `cdc-cms-service/config/config.go` — thêm prefix `CDS_`, dùng map `envBinds` + alias legacy.

**Việc cần làm** (Muscle sẽ thực hiện):

1. **Thêm prefix `CDS_`**:
   ```go
   v.SetEnvPrefix("CDS")
   v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
   v.AutomaticEnv()
   ```

2. **Thay `applyEnvOverrides` bằng `envBinds` map** (~50 entries, bao quát hết field):
   ```go
   envBinds := map[string][]string{
       "server.port":              {"CDS_SERVER_PORT"},
       "server.mode":              {"CDS_SERVER_MODE"},
       "systemDb.host":            {"CDS_SYSTEM_DB_HOST", "SYSTEMDB_HOST"},
       "systemDb.database":        {"CDS_SYSTEM_DB_DATABASE", "SYSTEMDB_DATABASE"},
       "systemDb.username":        {"CDS_SYSTEM_DB_USERNAME", "SYSTEMDB_USER"},
       "systemDb.password":        {"CDS_SYSTEM_DB_PASSWORD", "SYSTEMDB_PASSWORD"},
       "masterDb.host":            {"CDS_MASTER_DB_HOST", "MASTERDB_HOST"},
       "masterDb.database":        {"CDS_MASTER_DB_DATABASE", "MASTERDB_DATABASE"},
       "masterDb.username":        {"CDS_MASTER_DB_USERNAME", "MASTERDB_USER"},
       "masterDb.password":        {"CDS_MASTER_DB_PASSWORD", "MASTERDB_PASSWORD"},
       "shadowDb.host":            {"CDS_SHADOW_DB_HOST", "SHADOWDB_HOST"},
       // ... shadowDb full block tương tự
       "nats.url":                 {"CDS_NATS_URL", "NATS_URL"},
       "redis.url":                {"CDS_REDIS_URL", "REDIS_URL"},
       "jwt.secret":               {"CDS_JWT_SECRET", "JWT_SECRET"},
       "kafka.brokers":            {"CDS_KAFKA_BROKERS", "KAFKA_BROKERS"},
       "kafka.schemaRegistryUrl":  {"CDS_KAFKA_SCHEMA_REGISTRY_URL", "KAFKA_SCHEMA_REGISTRY_URL"},
       "debezium.kafkaConnectUrl": {"CDS_DEBEZIUM_KAFKA_CONNECT_URL", "KAFKA_CONNECT_URL"},
       "debezium.connectorName":   {"CDS_DEBEZIUM_CONNECTOR_NAME", "DEBEZIUM_CONNECTOR_NAME"},
       "masterKey":                {"CDS_MASTER_KEY", "MASTER_KEY"},
       "otel.endpoint":            {"CDS_OTEL_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_ENDPOINT"},
       "worker.poolSize":          {"CDS_WORKER_POOL_SIZE", "WORKER_POOL_SIZE"},
       "worker.batchSize":         {"CDS_WORKER_BATCH_SIZE", "BATCH_SIZE"},
       "worker.batchTimeout":      {"CDS_WORKER_BATCH_TIMEOUT", "BATCH_TIMEOUT"},
       // ...
   }
   for key, envs := range envBinds {
       _ = v.BindEnv(append([]string{key}, envs...)...)
   }
   ```
   → Manifest hiện tại với `WORKER_POOL_SIZE`, `BATCH_SIZE`, `BATCH_TIMEOUT` **vẫn ăn được** nhờ alias, **KHÔNG cần đổi tên ENV trong manifest**.

3. **Giữ riêng `ConnectionOverrides` map** (dynamic scan):
   - Tách block scan `CONNECTION_OVERRIDE_*` ra hàm riêng `applyConnectionOverrides(cfg)`.
   - Chạy SAU `v.Unmarshal(cfg)` và SAU `validateConfig`.
   - Có thể rename prefix → `CDS_CONNECTION_OVERRIDE_` cho nhất quán.

4. **Xóa `applyEnvOverrides`** cũ — toàn bộ key chuyển vào `envBinds`.

5. **Manifest k8s**: chỉ cần bổ sung 3 nhóm bắt buộc (SYSTEM_DB_*, MASTER_DB_*, JWT_SECRET) + pin image. Tên cũ giữ nguyên nhờ alias → migration không phá môi trường đang chạy.

6. **Update README** section "Environment variables" theo pattern mới.

**Ưu**:
- Pattern đã validated trên cdc-cms-service ("vẫn ăn bình thường") → low risk, copy gần như nguyên xi.
- **Alias legacy** → zero-downtime migration; manifest dev/staging hiện tại vẫn chạy.
- ENV name tường minh, snake-case dễ đoán → platform/security review pass.
- 2 codebase (CDS + CMS) đồng nhất pattern → lesson share, bảo trì rẻ.
- Audit dễ: 1 map duy nhất chứa toàn bộ ENV.

**Nhược**:
- Phải sửa Go code (~70 dòng) — Muscle scope, không phải Brain.
- Vẫn cần điều chỉnh manifest (thêm SYSTEM_DB_*, MASTER_DB_*, JWT_SECRET) — bắt buộc vì `validateConfig` cần.
- Cần test integration sau khi đổi (CMS đã làm rồi nên rủi ro thấp).

---

## Khuyến nghị

- **Nếu cần deploy gấp**: Hướng 1 (1-2 giờ, không đụng Go code).
- **Nếu platform team yêu cầu prefix**: Hướng 2 (0.5-1 ngày, đụng Go code → Muscle scope).
- **Nếu repo sắp lên production lâu dài**: Hướng 3 (1-2 ngày, đúng chuẩn).
- **Để trả lời câu hỏi "có nên chuyển về kiểu cdc-cms-service không"**: **Hướng 4** ⭐ — đáp án CÓ. Pattern CMS có 3 cơ chế mà CDS đang thiếu (prefix, envBinds map, multi-alias), và đã validated trên k8s.

Combo tối ưu: **Hướng 4 + Hướng 3** — Hướng 4 giải bài toán "k8s ăn được"; Hướng 3 giải bài toán multi-env / immutable manifest. Hai hướng KHÔNG đột sung nhau.

---

## Câu hỏi cần user trả lời trước khi Muscle thực thi

1. Platform team đang yêu cầu cụ thể điều gì? (tag pin? prefix CDS_? Helm chart? ConfigMap?)
2. Có ràng buộc time-to-deploy (gấp / không gấp)?
3. Đã có Secret `cdc-secrets` trong cluster chưa? Key bên trong là gì?
4. Image registry pull policy (tag cố định / digest pin)?
5. Có cần multi-env (dev/staging/prod) hay chỉ 1 env?
