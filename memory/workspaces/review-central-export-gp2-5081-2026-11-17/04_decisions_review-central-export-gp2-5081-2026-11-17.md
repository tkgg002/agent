# 04 — Decisions (Append ONLY)

- [2026-11-18T10:03:03+07:00] [Agent:laya_gate] DG1 S-IMPACT-1 FLAG resolved (DG1+DG2): Verified directly in GetAllTransHisExportHandler: const sortKey = Object.keys(sort || {})[0]; const sortWithTieBreaker = sort && !('_id' in sort) ? { ...sort, _id: (sort as any)[sortKey] === 1 ? 1 : -1 } : sort; creates a new shallow-cloned object without mutating query.sort. REF: /Users/trainguyen/Documents/work/centralized-export-service/domain/handlers/trans-his/GetAllTransHisExportHandler.ts:22-25
