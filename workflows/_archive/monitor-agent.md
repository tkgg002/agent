---
description: Passive Monitoring workflow - Giám sát Muscle mà không can thiệp, chỉ dọn dẹp resource
---

# Monitor Agent Workflow

> Quy tắc #5 (Giám sát không Can thiệp - Passive Monitoring)
> Quan sát. Dọn dẹp. Can thiệp tối thiểu (chỉ khi chiến lược sai).

## Khi nào dùng

Dùng khi Brain đã dispatch task cho Muscle và cần theo dõi tiến độ.
Trigger: `/monitor-agent`, hoặc tự động sau `/brain-delegate` step 5.

## Workflow Steps

### 1. Check Task Status (Observe)

// turbo
```bash
# Kiểm tra command đang chạy
command_status <command_id>
```

Quan sát:
- Muscle đang ở step nào trong `/muscle-execute`?
- Có error nào xuất hiện không?
- Tiến độ so với thời gian ước lượng?

### 2. System Health Check

// turbo
```bash
# CPU và Memory usage
top -l 1 -s 0 | head -10

# Disk space
df -h /Users/trainguyen

# Node processes
ps aux | grep node | grep -v grep
```

### 3. Resource Cleanup (Maintenance)

Chỉ thực hiện khi cần:

// turbo
```bash
# Giải phóng RAM nếu > 80% usage
sudo purge

# Kill zombie processes nếu có
ps aux | grep -E "defunct|zombie" | grep -v grep
```

### 4. Intervention Decision

```
┌─────────────────────────────────────────┐
│ Muscle đang làm gì?                    │
├─────────────────────────────────────────┤
│                                         │
│ Đang code đúng hướng?                  │
│   → ✅ KHÔNG can thiệp. Tiếp tục      │
│     observe ở step 1.                   │
│                                         │
│ Đang code SAI hướng chiến lược?         │
│   → ⚠️ CAN THIỆP. Redirect Muscle     │
│     về đúng scope/approach.             │
│                                         │
│ Code logic khác với ý Brain?            │
│   → ✅ KHÔNG can thiệp.               │
│     Muscle có quyền chọn implementation │
│     details.                            │
│                                         │
│ Muscle bị stuck > 10 phút?             │
│   → ⚠️ Hỏi Muscle cần gì.            │
│     Cung cấp thêm context nếu cần.     │
│                                         │
│ System resource cạn kiệt?              │
│   → 🔧 Chạy cleanup (step 3).         │
│                                         │
└─────────────────────────────────────────┘
```

### 5. Intervention Template (khi CẦN can thiệp)

```markdown
## Brain Intervention

### Reason
<Tại sao can thiệp - phải là lý do chiến lược, KHÔNG PHẢI implementation detail>

### Current Direction
<Muscle đang làm gì>

### Correct Direction
<Nên làm gì thay thế>

### Context bổ sung
<Thông tin thêm giúp Muscle redirect>
```

## Monitoring Schedule

| Interval | Action |
|----------|--------|
| Mỗi 2-3 phút | Check task status (step 1) |
| Mỗi 10 phút | System health check (step 2) |
| Khi RAM > 80% | Resource cleanup (step 3) |
| Khi detect sai hướng | Intervention (step 4-5) |

## Golden Rules

1. **OBSERVE trước, ACT sau** — Luôn collect data trước khi quyết định
2. **KHÔNG can thiệp logic** — Muscle chọn cách implement, Brain chọn WHAT, không chọn HOW
3. **Cleanup proactive** — Đừng đợi system crash mới dọn dẹp
4. **Minimal intervention** — Can thiệp ít nhất có thể, mỗi lần can thiệp phải có lý do rõ ràng

## Definition of Done
- [ ] Muscle đã hoàn thành task (exit code 0) hoặc đã escalate lên Brain nếu stuck
- [ ] System health ổn định (RAM < 80%, không có zombie processes)
- [ ] Mọi intervention đều có lý do chiến lược — không can thiệp logic implementation
