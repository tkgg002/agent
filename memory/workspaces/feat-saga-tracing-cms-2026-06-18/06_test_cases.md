# 06 — Test Cases & Validation Plan

## SAGA Tests

### TC-S1: All steps pass
```go
func TestRunner_AllPass(t *testing.T) {
    log := zap.NewNop()
    order := []string{}
    runner := saga.New("test", log,
        saga.Step{Name: "step1", Execute: func(ctx context.Context) error { order = append(order, "step1"); return nil }},
        saga.Step{Name: "step2", Execute: func(ctx context.Context) error { order = append(order, "step2"); return nil }},
    )
    err := runner.Run(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, []string{"step1", "step2"}, order)
}
```

### TC-S2: Step 1 fails, no compensation needed
```go
func TestRunner_Step1Fail_NoCompensation(t *testing.T) {
    compensated := false
    runner := saga.New("test", zap.NewNop(),
        saga.Step{
            Name:       "step1",
            Execute:    func(ctx context.Context) error { return errors.New("step1 error") },
            Compensate: func(ctx context.Context) error { compensated = true; return nil },
        },
        saga.Step{Name: "step2", Execute: func(ctx context.Context) error { return nil }},
    )
    err := runner.Run(context.Background())
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "step1")
    assert.False(t, compensated) // step1 chưa execute xong → không compensate
}
```

### TC-S3: Step 2 fails, step 1 compensated
```go
func TestRunner_Step2Fail_Step1Compensated(t *testing.T) {
    compensated := false
    runner := saga.New("test", zap.NewNop(),
        saga.Step{
            Name:       "step1",
            Execute:    func(ctx context.Context) error { return nil },
            Compensate: func(ctx context.Context) error { compensated = true; return nil },
        },
        saga.Step{
            Name:    "step2",
            Execute: func(ctx context.Context) error { return errors.New("step2 error") },
        },
    )
    err := runner.Run(context.Background())
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "step2")
    assert.True(t, compensated) // step1 đã done → cần compensate
}
```

### TC-S4: Compensation fails — không panic, log error
```go
func TestRunner_CompensationFails_NoP(t *testing.T) {
    runner := saga.New("test", zap.NewNop(),
        saga.Step{
            Name:       "step1",
            Execute:    func(ctx context.Context) error { return nil },
            Compensate: func(ctx context.Context) error { return errors.New("compensate error") },
        },
        saga.Step{Name: "step2", Execute: func(ctx context.Context) error { return errors.New("step2 error") }},
    )
    // Phải không panic, return error của step2
    assert.NotPanics(t, func() {
        err := runner.Run(context.Background())
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "step2")
    })
}
```

### TC-S5: Nil Compensate step là no-op
```go
func TestRunner_NilCompensate_NoOp(t *testing.T) {
    runner := saga.New("test", zap.NewNop(),
        saga.Step{Name: "step1", Execute: func(ctx context.Context) error { return nil }, Compensate: nil},
        saga.Step{Name: "step2", Execute: func(ctx context.Context) error { return errors.New("fail") }},
    )
    assert.NotPanics(t, func() {
        runner.Run(context.Background())
    })
}
```

---

## TRACING Tests

### TC-T1: OtelPropagator extract traceparent header
```go
func TestOtelPropagator_ExtractsTraceParent(t *testing.T) {
    // Setup in-memory trace exporter
    // Send request với header traceparent: 00-{traceID}-{spanID}-01
    // Assert: span trong context có đúng traceID
}
```

### TC-T2: CommandBus.Execute tạo span đúng tên
```go
func TestCommandBus_Execute_CreatesSpan(t *testing.T) {
    // Setup: sdktrace.NewSimpleSpanProcessor + tracetest.NewInMemoryExporter
    // Execute command → assert span name = "command_bus.execute"
    // Assert attribute command.type = cmd.Type()
}
```

### TC-T3: Existing CommandBus tests không regress
```bash
go test ./internal/infra/messaging/... -v -count=1
```

### TC-T4: EndSpan ghi error khi *err != nil
```go
func TestEndSpan_RecordsError(t *testing.T) {
    exporter := tracetest.NewInMemoryExporter()
    // ... setup tracer với exporter
    ctx, span := tracer.Start(context.Background(), "test")
    someErr := errors.New("test error")
    observability.EndSpan(span, &someErr)
    
    spans := exporter.GetSpans()
    assert.Len(t, spans, 1)
    assert.Equal(t, codes.Error, spans[0].Status.Code)
}
```

---

## Integration Validation

### IV-1: Full flow với go build
```bash
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
go build ./...        # EXIT=0
go vet ./...          # EXIT=0
go test ./... -count=1  # PASS
```

### IV-2: Saga log output
Khi chạy service và gọi `POST /api/v1/masters/{name}/approve` với NATS down:
```
DEBUG  saga.step.execute  {"saga": "master.approve", "step": "approve-schema-tx"}
DEBUG  saga.step.execute  {"saga": "master.approve", "step": "publish-master-create"}
ERROR  saga.step.failed   {"saga": "master.approve", "step": "publish-master-create", "error": "nats not ready"}
WARN   saga.compensate    {"saga": "master.approve", "step": "approve-schema-tx"}
```

### IV-3: Span hierarchy (khi OTel enabled)
```
command_bus.execute  [command.type=master.approve]
  └─ saga.master.approve  [saga.name=master.approve]
       ├─ saga.step  [saga.step=approve-schema-tx]
       └─ saga.step  [saga.step=publish-master-create]
```
