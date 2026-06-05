# 00_context.md — Bối cảnh

## Câu hỏi user

> `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/config/config.go` hiện tại trên k8s ko approve kiểu này

## Ngữ cảnh

- Repo: `centralized-data-service` (Go service, Viper-based config).
- `config.go` (313 dòng) dùng pattern:
  - `v.AutomaticEnv()` + `SetEnvKeyReplacer(".", "_")` → tự derive ENV name từ struct path.
  - `applyEnvOverrides(cfg)` hard-code 9 ENV key sau Unmarshal.
  - Quét động `CONNECTION_OVERRIDE_<NAME>` từ `os.Environ()` (comment ghi "dev/local only").
  - `validateConfig` bắt buộc: `Server.Port`, `SystemDB`, `MasterDB`, `JWT.Secret`, + cấm placeholder ở mode `production`.
- Manifest k8s duy nhất: `deployments/k8s/cdc-worker-deployment.yaml` (82 dòng, 1 Deployment + 1 HPA, 6 ENV vars).
- KHÔNG có Helm chart, KHÔNG có Kustomize overlay, KHÔNG có ConfigMap riêng.

## Scope

- Brain Code Prohibition (CLAUDE.md §12): chỉ phân tích & đề xuất, KHÔNG sửa `.go`/`.yaml`.
- Deliverable: `10_gap_analysis.md` (mismatch chi tiết) + `09_tasks_solution_proposal.md` (đề xuất hướng đi, đợi user approve).

## Non-goals

- Không refactor `config.go`.
- Không viết Helm chart / Kustomize / ConfigMap.
- Không sinh manifest mới.
