# 06_validation — Bằng chứng vật lý (Rule 16 G8)

## Build (G3)
- `go build ./...` (centralized-data-service) = **Success**, exit 0.
- Binary: `go build -o /tmp/cdc-worker-recon-p4h ./cmd/worker/` OK (49.7M).

## Red → Green (G2) — FIX 1 stale/orphan reaper
| Mốc | Trạng thái |
|---|---|
| RED (trước) | log worker cũ tích luỹ **55** warn `cannot resolve source URI` + **72** lỗi `recon_runs_one_running` (23505); run 'running' orphan tích luỹ qua nhiều instance_id cũ (mỗi worker restart bỏ lại). |
| Hành động | Deploy binary fix (p4h). SIGKILL live worker p4j → để lại **3 orphan** 'running'. |
| GREEN (p4h startup) | log: `reaped orphan running recon_runs from dead instances cancelled=3 current_instance=...6d5b6000` → **dọn cả 3 NGAY** (instance-based, không chờ 15'). |
| DB sau | `SELECT count(*) running, count(*) FILTER(started_at<now()-15min) FROM recon_runs WHERE status='running'` = **0 | 0**. Rows cancelled mang error_message `orphan from previous worker instance reaped at startup` (=4 qua 2 lần test). |

→ Đúng kịch bản bug (restart orphan <15' chặn table) được fix realtime. Không kill nhầm run live (instance hiện tại lúc startup chưa có run).

## FIX 2 — hết warn default_master
- Log worker fix (p4h): `grep -c "cannot resolve source URI"` = **0** (trước: 55). default_master (0 source bind) bị skip khỏi loop resolve source-URI. Connection có nguồn lỗi thật vẫn warn (giữ R2.3).

## Stability / functional (G3)
- `:8082` owner = p4h PID 17113; health=**200**; uptime ổn định; RSS ~39MB.
- `command listeners registered subjects_count=18` → worker hoạt động đầy đủ.
- fatal/panic trong log fix = **0** (bind :8082 OK).
- 23505/beginRun failed trong log fix = **0**.

## Negative-path / edge (G4)
- Reaper age-based >15' KHÔNG đụng run <15' (live). Instance-based startup chỉ cancel `instance_id IS DISTINCT FROM current` (gồm cả NULL legacy) — instance hiện tại vừa boot chưa có run nên không tự giết.
- SQL parameterized (`?` cho InstanceID) — không injection. error_message là chuỗi tĩnh; không log secret/PII.

## Ngoài scope (pre-existing, KHÔNG do fix này — flag cho user)
- `schema_adapter.go:229 gorm exec error: column "id" named in key does not exist` — có sẵn trong log p4j/p4g CŨ (2 lần mỗi log). Vấn đề schema-adapter riêng, không liên quan recon/registry.

## Deploy note
- Live worker hiện chạy `/tmp/cdc-worker-recon-p4h` (build từ source đã sửa). Source tree đã chứa thay đổi → hệ thống build/deploy ngoài của user sẽ pick-up ở lần build kế. Rollback: binary cũ `/tmp/cdc-worker-recon-p4j` còn trên disk. KHÔNG commit/push.
