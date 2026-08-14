# TR-069 Log File ID and List Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace artifact UUIDs with auto-increment numeric file IDs and simplify the GVA log-file page to exact SerialNumber filtering and user-relevant columns.

**Architecture:** MySQL allocates `model.Artifact.ID` before storage begins; the numeric ID is used by transfer events, object keys, audit, query, and download. Internal status and SHA-256 remain for reliability, while the management list exposes only available LOG files and minimal business metadata.

**Tech Stack:** Go, GORM, Gin, MySQL/SQLite tests, Vue 3, Element Plus, Node test runner, MinIO.

## Global Constraints

- Do not retain or expose an artifact UUID.
- Keep Task ID, Command ID, and CommandKey semantics unchanged.
- List only `AVAILABLE` LOG files and filter by complete SerialNumber equality.
- Keep SHA-256 and artifact status internally for integrity, reconciliation, and retention.
- Keep JWT, Casbin, device-scope checks, and backend-streamed downloads.
- Preserve bounded streaming and do not buffer file bodies.

---

### Task 1: Numeric file identity and transfer lifecycle

**Files:**
- Modify: `server/plugin/tr069/model/transfer.go`
- Modify: `server/plugin/tr069/service/artifact_store.go`
- Modify: `server/plugin/tr069/service/transfer_store.go`
- Modify: `server/plugin/tr069/service/transfer_receiver.go`
- Modify: `server/plugin/tr069/service/transfer_lifecycle.go`
- Modify: `server/plugin/tr069/service/transfer_worker.go`
- Test: `server/plugin/tr069/service/transfer_store_test.go`
- Test: `server/plugin/tr069/service/transfer_receiver_test.go`
- Test: `server/plugin/tr069/service/transfer_lifecycle_test.go`
- Test: `server/plugin/tr069/service/transfer_worker_test.go`
- Test: `server/plugin/tr069/service/artifact_store_contract_test.go`

**Interfaces:**
- Produces: `Artifact.ID uint64`, `TransferEvent.FileID uint64`, `ArtifactObjectKey(..., fileID uint64)`, and numeric `MarkArtifactAvailable`, `MarkArtifactFailed`, and `OnArtifactAvailable` parameters.

- [x] Write failing tests that require auto-increment file IDs, numeric event linkage, and object keys ending in the decimal file ID.
- [x] Run `cd server && env GOWORK=off go test ./plugin/tr069/service -count=1` and confirm compile or assertion failures reference the old UUID API.
- [x] Replace artifact UUID storage and lifecycle references with numeric file IDs while leaving task UUIDs unchanged.
- [x] Re-run the service tests and confirm they pass.

### Task 2: Minimal list and numeric download API

**Files:**
- Modify: `server/plugin/tr069/model/request/artifact.go`
- Modify: `server/plugin/tr069/model/response/artifact.go`
- Modify: `server/plugin/tr069/api/artifact.go`
- Modify: `server/plugin/tr069/router/artifact.go`
- Modify: `server/plugin/tr069/initialize/api.go`
- Modify: `server/plugin/tr069/middleware/download_audit.go`
- Test: `server/plugin/tr069/api/artifact_test.go`
- Test: `server/plugin/tr069/middleware/download_audit_test.go`

**Interfaces:**
- Consumes: numeric `Artifact.ID` and `TransferStore.GetAvailableArtifact(ctx, fileID uint64)`.
- Produces: `GET /tr069/artifact/list?...&serialNumber=<exact>` and `GET /tr069/artifact/:fileId/download`.

- [x] Write failing API and audit tests for alphanumeric exact SerialNumber filtering, available-only rows, minimal DTO fields, and numeric download IDs.
- [x] Run `cd server && env GOWORK=off go test ./plugin/tr069/api ./plugin/tr069/middleware -count=1` and confirm failures describe the old request, DTO, and route contract.
- [x] Implement exact SerialNumber equality, fixed LOG/AVAILABLE constraints, numeric lookup, and numeric audit metadata.
- [x] Re-run API and middleware tests and confirm they pass.

### Task 3: Simplified GVA log-file page

**Files:**
- Modify: `web/src/plugin/tr069/api/log-file.js`
- Modify: `web/src/plugin/tr069/view/log-file/index.vue`
- Modify: `web/src/plugin/tr069/view/log-file/log-file-view.js`
- Modify: `web/src/plugin/tr069/view/log-file/log-file-view.test.js`
- Modify: `web/src/plugin/tr069/view/log-file/log-file.contract.test.js`

**Interfaces:**
- Consumes: rows containing `fileId`, `serialNumber`, `originalName`, `source`, `size`, and `receivedAt`.
- Produces: a GVA-themed list with exact text SerialNumber filtering and numeric file-ID downloads.

- [x] Write failing frontend tests requiring `el-input`, `serialNumber`, `fileId`, MB formatting, and absence of status, SHA-256, OUI, and database-device-ID UI.
- [x] Run `cd web && node --test src/plugin/tr069/view/log-file/*.test.js` and confirm the contract fails against the old page.
- [x] Implement the minimal page and API wrapper changes.
- [x] Run the frontend tests and `cd web && npm run build`.

### Task 4: Development reset and real verification

**Files:**
- Modify: `openspec/changes/add-tr069-log-collection/tasks.md`

**Interfaces:**
- Consumes: final MySQL models, MinIO driver, GVA server, and BS upload stack.
- Produces: freshly recreated transfer tables and a real downloadable BS artifact with numeric file ID.

- [x] Stop GVA and prevent uploads while resetting `tr069_transfer_events`, `tr069_artifacts`, `tr069_transfer_tasks` and the configured MinIO LOG prefix.
- [x] Start GVA so AutoMigrate creates the final schema, then restart the frontend.
- [x] Trigger or observe a real BS upload and verify HTTP 201, numeric file ID, exact `SNB123456789` filtering, MinIO object presence, and byte-identical download.
- [x] Run `git diff --check`, full TR-069 tests, `go vet`, frontend tests/build, and strict OpenSpec validation.
