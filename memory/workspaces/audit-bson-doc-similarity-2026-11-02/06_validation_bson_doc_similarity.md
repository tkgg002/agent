---
tier: T2
suffix: bson_doc_similarity
workspace: audit-bson-doc-similarity-2026-11-02
---

# 06. Validation Evidence — `audit-bson-doc-similarity-2026-11-02`

## 1. Biên lai Kiểm thử Unit & Integration trên MongoDB Thật (`EV-...`)
- **Biên lai 1 (`EV-20261102T130838-d40b` — Live MongoDB Integration Test):**
  - Lệnh: `go test -count=1 -v -run TestMongoOCCBulkModels_ShadowWithoutUpdatedAtAndPruneLeakedSourceFieldValues_LiveMongo ./internal/service/master`
  - Kết quả: `PASS` (`EV-20261102T130838-d40b`, `exit=0`).
- **Biên lai 2 (`EV-20261102T130845-82c5` — Unit & BSON Coercion Test):**
  - Lệnh: `go test -count=1 -v -run TestApplySourceFieldTypesToBSONDoc ./internal/service/master`
  - Kết quả: `PASS` (`EV-20261102T130845-82c5`, `exit=0`), kiểm chứng `value` kiểu `NUMERIC` nguyên trong dải `int32` được thu hẹp về `int32` (`NumberInt`), và số thập phân `9999.50` giữ nguyên `primitive.Decimal128` (`NumberDecimal`).
- **Biên lai 3 (`EV-20261102T130850-3c8d` — Runtime Listener Verification):**
  - Lệnh: `lsof -nP -iTCP:8082 -sTCP:LISTEN`
  - Kết quả: `PASS` (`EV-20261102T130850-3c8d`, `exit=0`, `cds-worke PID 5148 TCP *:8082 LISTEN`).
