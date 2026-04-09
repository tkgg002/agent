# 04_decisions.md - Decision Log

## ADR-001: 7-File Structure
- **Context**: User found 2-file structure (`active_plan`, `progress`) insufficient for resuming context.
- **Decision**: Adopt 7-file structure to strictly separate Concerns (Requirements vs Plan vs Implementation).
- **Consequences**:
    - (+) Richer context.
    - (-) More file management overhead (Brain needs to update more files).
    - (*) Mitigation: Update workflow to automate scaffolding.

## ADR-002: Explicit Signature
- **Context**: User noted lack of role clarity.
- **Decision**: Every response must end with Role/Task/Exec/Skills signature.
