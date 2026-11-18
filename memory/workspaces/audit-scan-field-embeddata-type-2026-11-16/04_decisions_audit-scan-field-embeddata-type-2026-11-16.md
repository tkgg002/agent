# 04 — Decisions (Append ONLY)

- [2026-11-16T15:12:10+07:00] [Agent:laya_gate] DG1 S-RC1 FLAG resolved: Đã kiểm chứng: vòng for k, v := range doc gán fieldMap[k] = v không điều kiện, doc sau đè doc trước. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/mongo_introspection.go:137-148
- [2026-11-16T15:12:10+07:00] [Agent:laya_gate] DG1 S-IMPACT FLAG resolved: Đã kiểm chứng: sau nhánh len==1, if _, ok := normalizedSet["JSONB"]; ok { return "JSONB" }. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/discover_handler_mongo.go:123-131
