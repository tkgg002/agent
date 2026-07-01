# Plan - ReconSelfHealing

## Execution Phases
1. **Research & Design**: Inspect how `TransmuterModule.Run` handles master upsert and soft-deletes.
2. **Implementation**: Integrate orphan identification based on `onlySourceIDs` scope and build update query to batch update orphans.
3. **Verification**: Run standard in-memory test databases mimicking PG-specific syntax through dialect adapters, verifying correctness.
4. **Strict Audit Fixes**: Apply fixes for 3 critical logic issues (watermark fallback, orphan skip in heal, transmuter pagination data deletion risk) and 1 batch logging visibility issue.

