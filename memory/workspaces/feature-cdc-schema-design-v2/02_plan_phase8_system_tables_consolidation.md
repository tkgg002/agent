# Plan — Phase 8 System Tables Consolidation

1. Đổi model system tables sang cdc_system.
2. Thêm migration dời bảng cũ từ public/cdc_internal sang cdc_system.
3. Vá raw SQL runtime cho các bảng system chính.
4. Đổi sinkworker shadow namespace sang shadow_<source_db>.
5. Verify compile/test và audit leftover references.
