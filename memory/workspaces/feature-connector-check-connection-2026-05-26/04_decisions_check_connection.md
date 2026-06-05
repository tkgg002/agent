# 04_decisions_check_connection — ADR Log

> **Phase**: `check_connection`
> **Format**: ADR-XXX (Architecture Decision Record)

---

## ADR-001: Scope = MongoDB-only P0, defer MySQL/Postgres

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

Audit cho thấy:
- Mongo: 70% infra đã có (worker service + BE endpoint + NATS subject). Gap = handler accept full URI + FE.
- MySQL/PG: 0% infra. Cần implement service từ đầu.

### Decision

P0 = Mongo. MySQL + PG defer phase sau (`connector-check-mysql`, `connector-check-pg`).

### Rationale

- §6 Simplicity: ship value sớm với 1 DB type.
- MySQL/PG cần thiết kế interface trước (xem ADR-007 future).
- User original use case là Mongo connector — match scope.

### Consequences

- ✅ Ship được trong 6-8h.
- ⚠️ User MySQL/PG vẫn dùng pattern cũ (text input collection.include.list / table.include.list). Chấp nhận.
- ⚠️ Khi mở rộng MySQL → cần thiết kế interface để tránh copy-paste 3 lần.

---

## ADR-002: BE DTO shape — POST với body có `uri` (giữ GET legacy)

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

Existing endpoints là GET với query params `host`, `port`. Mongo URL chứa password → GET sẽ leak vào access log nginx / browser history / proxy log.

### Options

| # | Option | Pros | Cons |
|---|---|---|---|
| A | GET với `?uri=` URL-encoded | Đơn giản | URI có password leak vào access log/browser history. Security risk. |
| B | POST với body JSON `{uri}` | Body không log vào access log mặc định. Auth header dễ. | Hơi inconsistent với GET hiện tại |
| C | Custom header `X-Mongo-URI` | Header không log default | Lạ, không discoverable, khó test với curl |
| D | Encrypt URI client-side trước GET | Defense in depth | Quá over-engineer cho P0 |

### Decision

**B** — POST với body JSON. Giữ GET legacy.

### Rationale

- Security là yêu cầu cứng (N7, L-3275).
- POST body không log mặc định vào access log → KHÔNG leak.
- Add POST route mới, KHÔNG xóa GET → backward compat (R12).
- Nếu legacy GET caller có audit log leak password → đó là vấn đề cũ, không tạo regression mới.

### Consequences

- ✅ Security mặc định.
- ⚠️ FE phải dùng POST mới — clearly documented.
- ⚠️ Test cũ với GET vẫn pass.

---

## ADR-003: Error UX — Map 5-case IntrospectDiagnosis sang VN message

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

Worker service đã có `IntrospectDiagnosis` (status: `ok / cluster_err / db_missing / coll_missing / empty / no_fields`). Cần map sang user-facing VN message.

### Decision

5-case mapping:

| Status | VN message | Action hint |
|---|---|---|
| `ok` | (không hiển thị, show multi-select) | — |
| `cluster_err` | "Không kết nối được tới Mongo. Kiểm tra URL và đảm bảo Mongo đang chạy." | "Mở terminal: `mongosh '<sanitized_dsn>'` để test" |
| `auth_err` (sub-case parse từ error msg) | "Sai thông tin xác thực. Kiểm tra user/password trong URL." | — |
| `db_missing` | "Database `<X>` không tồn tại. Database có sẵn: `<list top 50>`." | "Click để chọn database từ danh sách." |
| `empty` | "Database `<X>` chưa có collection nào." | "Tạo collection rồi check lại." |
| `coll_missing` | (N/A cho UC list collections, chỉ trigger ở scan-fields) | — |
| `no_fields` | (N/A cho UC list collections) | — |
| `timeout` | "Worker không phản hồi sau 10s. Vui lòng thử lại." | — |
| `unknown` | "Lỗi không xác định: `<error>`." | "Liên hệ ops với error code." |

### Rationale

- User là Ops admin VN, message VN giúp debug nhanh.
- Available list giúp self-service fix khi typo.
- Action hint là optional, không bắt buộc P0.

### Consequences

- ✅ Diagnostic fidelity (L-2026-05-19).
- ⚠️ Worker phải fill `available_databases` ở response — check service signature có support.

---

## ADR-004: Multi-select default = ALL collections selected

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

User explicit: "ban đầu auto chọn hết". Cũng match yêu cầu workspace cũ "để trống = CDC all".

### Decision

Sau check PASS:
1. Set form field `collectionNames` = `response.collections` (full list).
2. Multi-select render với tất cả option pre-selected.
3. User có thể uncheck tùy ý trước Create.

### Rationale

- Match user spec.
- Default safe (capture all) — uncheck là action có ý thức.
- Consistent với BE behavior cũ (empty = all).

### Consequences

- ✅ Submit payload luôn explicit `collection.include.list = a,b,c` (không còn empty).
- ⚠️ Connector cũ có `collection.include.list = ""` (empty) sẽ render multi-select với selected = [] sau check (vì FE không gọi check lúc edit). Cần ADR-006 handle edit existing case.

---

## ADR-005: Progress UX — Spin indeterminate + step label, KHÔNG Progress bar %

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

User nói "có progress bar chạy". Nghĩa đen = `<Progress percent={X}>`. Nhưng:
- Mongo introspect là sync 1 RTT, không có % chia nhỏ được.
- Faking progress (vd interval increment 10%) = lừa user (anti-pattern UX).

### Options

| # | Option | Pros | Cons |
|---|---|---|---|
| A | `<Progress percent>` real % (cần streaming) | Match user spec literal | Phải implement WebSocket/SSE — over-engineer P0 |
| B | `<Spin>` indeterminate + step label | Honest, simple | Không có % cụ thể |
| C | `<Progress percent>` fake interval | Match visual | Lừa user, anti-pattern |
| D | `<Steps>` Antd component với 3 step (Connect → List DB → List Coll) | Visible progression | Over-design cho UC sync |

### Decision

**B** — `<Spin>` + step label dạng:
```
⏳ Đang kiểm tra kết nối...    [step 1/2: Kết nối Mongo]
⏳ Đang liệt kê collections... [step 2/2: Liệt kê collections]
```

Implementation: 2 spinner update qua state `checkStep: 'connecting' | 'listing'`. Worker reply 1 lần nhưng FE có thể chia 2 step visual nếu cần (trong response time ~1-2s).

### Rationale

- Honest UX, không fake.
- User intent "progress bar" thực tế là "feedback rằng đang chạy" — Spin đã đủ.
- Có thể upgrade thành streaming progress trong future phase.

### Consequences

- ✅ Honest, simple.
- ⚠️ User có thể phản hồi "muốn progress %". Nếu vậy → future phase implement SSE streaming.

---

## ADR-006: Gate Create button — disable until check PASS + invalidate on URI/DB change

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

R7 + R8 yêu cầu Create chỉ enable khi check PASS. R8 yêu cầu đổi URI/DB → invalidate.

### Decision

```
disabled = (mode === 'create') && (
  !checkResult ||
  checkResult.status !== 'ok' ||
  uriChanged ||
  dbChanged
)
```

Edit existing connector: bypass check requirement — load existing config, multi-select pre-fill, KHÔNG ép re-check (user có thể optional click Check để verify).

### Rationale

- Create mode: bắt buộc gate để tránh tạo connector lỗi.
- Edit mode: existing connector đã work — không ép re-check (UX friction).
- Watch URI/DB change để invalidate stale check result.

### Consequences

- ✅ Match user spec.
- ⚠️ Edge case: user edit URI rồi save mà KHÔNG check → BE vẫn validate. Acceptable trade-off.

---

## ADR-007 (Forward-looking): Cross-DB driver interface — defer

**Date**: 2026-05-26
**Status**: 📋 DEFERRED (future phase `connector-driver-abstraction`)

### Context

Khi implement MySQL/PG check sau này, sẽ có 3 service struct tương tự `MongoIntrospectionService`. Có nên thiết kế interface `SourceDriver` ngay?

### Decision

**DEFER**. P0 = Mongo only. Khi có MySQL pop up, sẽ refactor 2-driver thành interface với 2 implementation. Lúc đó pattern rõ.

### Rationale

- YAGNI: design interface với 1 implementation = over-design.
- Pattern thực tế rút ra từ 2-3 implementation, không phải hypothesis.
- Refactor lúc có 2 driver = chi phí thấp (extract interface trong Go là mechanical).

### Consequences

- ⚠️ MySQL/PG phase sau sẽ tốn 1-2h refactor extract interface.
- ✅ Tránh design "vọng tưởng" giờ.

---

## ADR-008: KHÔNG dùng wizard session table cho UC này

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

Codebase có `cdc_wizard_sessions` table + endpoints `POST/PATCH /v1/wizard/sessions`. Có nên persist check result vào session?

### Decision

**KHÔNG**. UC này ephemeral — React state đủ.

### Rationale

- Wizard session = multi-step persist (user có thể quay lại sau hours/days).
- Check Connection = 1-shot sync, dùng ngay trong modal.
- Add session = thêm latency + complexity + clean-up burden.

### Consequences

- ✅ Simple.
- ⚠️ Refresh browser mid-flow → mất check result, user check lại. Acceptable.

---

## ADR-009: SUPERSEDE workspace cũ `feature-connector-default-collections-2026-05-25`

**Date**: 2026-05-26
**Status**: ✅ ACCEPTED

### Context

Workspace cũ propose "FE-only hint" approach. Workspace mới là full feature thay thế.

### Decision

Mark workspace cũ là **SUPERSEDED** trong `active_plans.md`. Doc cũ giữ nguyên làm lịch sử quyết định.

### Rationale

- Workspace mới giải quyết cùng problem (collections default behavior) với UX explicit hơn.
- Workspace cũ vẫn có value lịch sử (ADR-001 cũ giải thích vì sao chọn FE-only — bị supersede bằng audit mới).

### Consequences

- ✅ History tránh xóa, audit-friendly.
- ⚠️ Khi đọc lessons / active_plans phải biết phân biệt SUPERSEDED vs DONE.
