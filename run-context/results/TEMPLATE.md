# Audit & Execution Run Report Template

> **DOCUMENT TYPE**: EXECUTION & VERIFICATION AUDIT RECORD  
> **USAGE**: Clone this template for every execution cycle, save to `run-context/results/latest.md`, and archive to `run-context/results/history/YYYY-MM-DD_HH-mm_<run-name>.md`.  
> **LANGUAGE REQUIREMENT**: 100% EXCLUSIVE TECHNICAL ENGLISH

---

## 1. Run Metadata & Execution Scope

| Field | Value |
| :--- | :--- |
| **Run Identifier** | `RUN-YYYYMMDD-HHMM` |
| **Timestamp (Local)** | `YYYY-MM-DD HH:mm:ss UTC+7` |
| **Primary Objective** | Concise description of the execution objective and requirements |
| **Target Service(s)** | Specific microservice(s) or shared package (e.g., `order`, `auth`, `pkg/saga`) |
| **Execution Environment** | Host Windows PowerShell / WSL2 Ubuntu-22.04 Docker |
| **Lead Executor** | AI Agent / Antigravity |

---

## 2. Pre-Execution Verification Confirmation

- [ ] Reviewed [run-context/rule.md](file:///c:/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices/run-context/rule.md) (Strict English-only rule, safety constraints, WSL2 rules).
- [ ] Reviewed [run-context/context.md](file:///c:/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices/run-context/context.md) (Microservice topology, ports, datastore mappings).
- [ ] Reviewed [run-context/results/latest.md](file:///c:/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices/run-context/results/latest.md) (Previous execution status and known issues).
- [ ] Confirmed target scope: No unconstrained refactors outside the designated service.

---

## 3. Terminal Commands Executed

```powershell
# Exact, verbatim commands executed during this run
# Example:
# cd order
# gofmt -w .
# go vet ./...
# go test -v -race ./...
```

---

## 4. Code & Configuration Modifications

| Action | Relative File Path | Rationale & Impact Summary |
| :--- | :--- | :--- |
| `[NEW]` | `path/to/file` | Created new component or migration |
| `[MODIFY]` | `path/to/file` | Implemented business logic or bug fix |
| `[DELETE]` | `path/to/file` | Removed obsolete module or config |

---

## 5. Verification & Test Suite Outcome

### 5.1 Static Analysis & Formatting
- **Go Formatting (`gofmt -l .`)**: `PASS` (Clean)
- **Go Vet (`go vet ./...`)**: `PASS` (0 warnings)

### 5.2 Unit & Integration Tests
- **Target Module**: `<service-name>`
- **Test Command**: `go test -v -race ./...`
- **Output Summary**:
  ```text
  === RUN   Test...
  --- PASS: Test... (0.05s)
  PASS
  ok      github.com/TanFuc/FDemo-microservices/<service>  0.412s
  ```

### 5.3 Infrastructure & Docker Health (if applicable)
- **Compose Stack Status**:
  - PostgreSQL (Port `15432`): `HEALTHY`
  - Redis (Port `16379`): `HEALTHY`
  - MinIO / ClickHouse / MongoDB / RabbitMQ: `HEALTHY`

---

## 6. Issues Encountered, Root Cause & Resolutions

### Issue 1: [Short Title]
- **Symptom & Error Message**: Full stack trace or error log.
- **Root Cause Analysis**: Technical explanation of why the failure occurred.
- **Remediation Implemented**: Exact code change or command used to resolve it.

---

## 7. Open Regressions & Pending Work Items

- **Pending Tasks**: Specific remaining requirements for future runs.
- **Known Gotchas for Next Agent**: Specific warnings or context needed before touching these files again.

---

## 8. Final Status & Sign-off

- [ ] **Execution Status**: `COMPLETED` / `PARTIAL` / `FAILED`
- [ ] **Documentation Language**: Confirmed 100% English across all updated files.
- [ ] **Archive Confirmation**: Snapshot saved to `run-context/results/history/`.
