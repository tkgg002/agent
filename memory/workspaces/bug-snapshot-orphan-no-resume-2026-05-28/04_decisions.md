# 04_decisions — ADR Bug Snapshot Orphan No-Resume

## ADR-001 — Reclaim via demote-to-paused + re-publish

**Context**: Khi worker crash, row stuck `status='running'`. Có 3 cách reclaim:
1. Worker boot UPDATE status='paused' + publish NATS → claimProgress đi nhánh resume.
2. Worker boot publish NATS với field `force_reclaim=true` → claimProgress thêm nhánh bypass zombie window.
3. Worker boot tự gọi runSnapshot inline (không qua NATS).

**Decision**: Cách 1 (demote-to-paused + publish).

**Consequence**:
- (+) Tận dụng nhánh resume hiện có (line 637-655) — không sửa claimProgress.
- (+) NATS publish đi qua queue group → đúng worker hiện đang khỏe pick up (không bị buộc local worker).
- (+) Logging path đồng nhất với resume manual (UI).
- (-) 2 round-trip DB (UPDATE + claim re-SELECT). Trade-off acceptable cho 1 lần boot.

**Alternative rejected**:
- Cách 2: tăng phức tạp payload + claim logic. Defer.
- Cách 3: bypass queue group → khi nhiều worker, primary worker boot lấy hết. Mất load distribution.

---

## ADR-002 — `staleAfter = 60s` mặc định

**Context**: Checkpoint hiện UPDATE updated_at sau mỗi batch (~5s normal). Nếu worker chết, updated_at không tiến.

**Decision**: 60s default, configurable qua env `SNAPSHOT_STALE_AFTER_SECONDS`.

**Consequence**:
- (+) 12x buffer batch interval → false-reclaim risk thấp.
- (+) User chỉ chờ tối đa 60s sau worker restart để snapshot tiếp tục.
- (-) Nếu 1 batch flush > 60s (vd schema drift retry) sẽ false-reclaim → 2 worker cùng chạy. Mitigation: claimProgress DB transaction lock chống double-claim.

**Alternative rejected**:
- 30s: false-reclaim cao với batch lớn.
- 5 phút: user phải chờ quá lâu.

---

## ADR-003 — Boot reclaim async goroutine (non-blocking)

**Context**: Boot reclaim cần DB query. Nếu DB chậm/lỗi, worker startup bị block → tất cả subscribe trễ.

**Decision**: Spawn goroutine sau khi `QueueSubscribe` return success. Lỗi reclaim → log warn, không return err lên main.

**Consequence**:
- (+) Worker vẫn nhận message mới ngay cả khi reclaim chết.
- (+) Tránh circular dep: reclaim cần subscriber sẵn sàng để nhận lại publish.
- (-) Order non-deterministic: nếu new message tới trước reclaim publish, 2 message cho cùng row → claimProgress transaction lock chống.

**Alternative rejected**:
- Sync block startup: DB hiccup làm worker không start.
- Cron tick mỗi 1 phút: thêm scheduler infra; defer.

---

## ADR-004 — FE thresh 60s khớp BE

**Context**: FE quyết định nút Resume xuất hiện. Threshold lệch BE-FE sẽ confuse operator.

**Decision**: FE hardcode `STALE_RUNNING_THRESHOLD_MS = 60_000` (khớp BE default). Future: query backend `/api/v1/snapshot-progress/config` nếu cần dynamic.

**Consequence**:
- (+) Đơn giản, không thêm round-trip API.
- (-) Nếu BE đổi env thành 120s, FE vẫn show resume từ 60s → operator click resume → backend từ chối hoặc accept tùy claimProgress. Acceptable cho phase 1.

**Alternative rejected**:
- API endpoint trả config: thêm route + cache; defer Phase 2.
- WebSocket push config khi đổi: over-engineer.

---

## ADR-005 — Force Resume UI label khác Resume

**Context**: Nút Resume bình thường (status='paused') vs Resume cho stale running cùng action. Operator có thể không phân biệt risk.

**Decision**: Stale running button label = **"Force Resume"** + icon `<WarningOutlined />` + tooltip giải thích.

**Consequence**:
- (+) Operator awareness về risk.
- (+) Confirm dialog branch text rõ ràng.
- (-) Tăng visual noise. Acceptable.

**Alternative rejected**:
- Cùng nhãn Resume: operator không phân biệt được orphan vs paused thường.

---

## ADR-006 — KHÔNG dùng JetStream durable subscription

**Context**: JetStream durable subscription sẽ tự động re-deliver message khi worker restart, giải bài toán này tự nhiên.

**Decision**: KHÔNG migrate sang JetStream trong PR này. Giữ NATS core pub-sub. Boot reclaim làm patch tactical.

**Consequence**:
- (+) Patch tối thiểu §6, không refactor NATS layer.
- (-) JetStream là solution sạch hơn dài hạn. Defer roadmap.

---

## ADR-007 — Lesson global: "boot-time orphan scan for message-driven runners"

**Context**: Lesson `L-2026-05-28-mark-done-without-completeness-guard` đã enforce invariant ở edge. Bug hôm nay là lớp khác: process death không có graceful handoff.

**Decision**: Ghi lesson mới `L-2026-05-28-boot-reclaim-missing-for-message-driven-runner` Global Pattern: "Long-running job P consuming messages M from queue Q, on process P boot must scan persistent in-flight set S(P) for stale(s) > τ, re-publish M(s) or notify operator — else state(s) is permanently orphan until external trigger."

**Consequence**:
- Áp dụng được cho mọi message-driven worker (Kafka consumer, NATS subscriber, SQS, RabbitMQ).
- Liên quan: L-2026-05-28-mark-done-without-completeness-guard (cùng module snapshot, complement).
