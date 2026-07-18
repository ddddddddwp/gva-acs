## 1. Configuration and Data Foundation

- [ ] 1.1 Add failing configuration tests for enabled LOG ingress with empty credentials, invalid channel limits, invalid trusted-proxy ranges, and incomplete MinIO settings
- [ ] 1.2 Add typed `file-ingress`, channel, shared Basic/Digest authentication, trusted-proxy, identity-binding, and artifact-store configuration with secure startup validation
- [ ] 1.3 Add failing model tests for transfer-task, artifact, and transfer-event defaults, indexes, state constraints, and secret-free serialization
- [ ] 1.4 Implement the three TR-069 transfer models and register them in plugin AutoMigrate
- [ ] 1.5 Add repositories with transaction-safe creation, conditional state changes, device-filtered pagination, stale-receive scans, and event append operations

## 2. Inform Device Binding and Source Resolution

- [ ] 2.1 Add failing tests for direct client IP, trusted proxy forwarding, untrusted forwarded headers, IPv4/IPv6 normalization, binding expiry, and ambiguous source IPs
- [ ] 2.2 Refactor TR-069 client-IP extraction into a trusted-proxy-aware resolver shared by CWMP and file ingress
- [ ] 2.3 Refresh the Redis upload identity binding only after Inform device persistence succeeds, including device ID, OUI, ProductClass, SerialNumber, IP, Inform time, and configurable TTL
- [ ] 2.4 Implement MySQL fallback and Redis backfill that return a device only when the registered-device match is unique and recent
- [ ] 2.5 Implement active-task/source-IP/device-identity resolution with generic external rejection and specific secret-free internal failure events

## 3. Basic and Digest Authentication

- [ ] 3.1 Add PUT/POST protocol-vector tests for valid/invalid Basic, Digest qop=auth, method-sensitive digest calculation, MD5, MD5-sess, expired nonce, replayed nonce-count, missing credentials, and empty server configuration
- [ ] 3.2 Implement shared LOG Basic authentication with constant-time credential comparison and sanitized audit fields
- [ ] 3.3 Implement Digest challenge and verification with expiring nonces and Redis-backed replay protection
- [ ] 3.4 Compose route authentication so successful authentication grants only the configured channel and never derives a device from username

## 4. Streaming Artifact Storage

- [ ] 4.1 Define `ArtifactStore` and `ArtifactWriter` interfaces plus storage-neutral object metadata and typed errors
- [ ] 4.2 Add contract tests covering Begin/Write/Commit/Abort/Open/Stat/Delete, context cancellation, duplicate abort, and unavailable objects
- [ ] 4.3 Implement the MinIO/S3-compatible streaming driver using multipart upload without `multipart.FileHeader`, `bytes.Buffer`, or whole-file reads
- [ ] 4.4 Add an isolated in-memory or temporary-directory store for service and route tests
- [ ] 4.5 Implement deterministic opaque object keys and sanitized original-filename metadata without exposing provider fields through DTOs

## 5. LOG File Ingress Pipeline

- [ ] 5.1 Add failing route/service tests proving both `PUT /acs/log` and `POST /acs/log` accept raw file bodies and bypass RawDump/XML/multipart parsing, disabled PUT/POST `/acs/pm` and `/acs/mr` do not invoke LOG handling, and other methods return `405` with the correct Allow header
- [ ] 5.2 Implement the channel registry and mount PUT/POST for every enabled file route on the existing 7458 server with one shared route-specific middleware and handler chain
- [ ] 5.3 Implement global/channel/per-device admission control with `503` plus `Retry-After` and guaranteed token release
- [ ] 5.4 Implement receive metadata creation, fixed-buffer streaming, Content-Length early rejection, streaming size enforcement, SHA-256 calculation, upload timeout, cancellation Abort, and `204` success
- [ ] 5.5 Implement duplicate same-content idempotency, different-content conflict rejection, and secret/file-body-free structured events
- [ ] 5.6 Verify all authentication and identity failures occur before object creation and that no rejected request creates an available artifact

## 6. Active Upload and Transfer State Machine

- [ ] 6.1 Add failing state-machine tests for ACTIVE/PERIODIC classification, one waiting active task per device, response/file/TransferComplete arrival permutations, faults, timeout, duplicate messages, and ambiguous tasks
- [ ] 6.2 Create an ACTIVE transfer task transactionally when the existing Upload RPC command is accepted, using the configured shared URL and credentials while preserving CommandKey linkage
- [ ] 6.3 Associate a PUT or POST upload with the unique `WAITING_FILE` task or create a PERIODIC task when none exists
- [ ] 6.4 Implement idempotent completion for UploadResponse status 0 and status 1, including out-of-order successful TransferComplete and retained artifacts on faults
- [ ] 6.5 Integrate file-wait and existing TransferComplete timeout handling so every created transfer task reaches a defined terminal state

## 7. Reconciliation and Retention

- [ ] 7.1 Add failing tests for object-committed/database-incomplete, stale receive without object, checksum mismatch, repeated reconciliation, deletion failure, and repeated cleanup
- [ ] 7.2 Implement a stale `RECEIVING` reconciler that finalizes matching objects or removes residual data and marks failure
- [ ] 7.3 Implement retention scanning with conditional `DELETING`/`DELETED` transitions while preserving task and event audit metadata
- [ ] 7.4 Register worker lifecycle with plugin startup/shutdown and ensure concurrent service instances cannot process the same row twice

## 8. Management APIs and Permissions

- [ ] 8.1 Add failing service/API tests for pagination, exact device ID filtering, device data scope, unavailable artifact behavior, missing object behavior, JWT/Casbin denial, and secret/provider-field omission
- [ ] 8.2 Implement artifact list DTO/service/API with device identity metadata and exact `deviceId` filtering
- [ ] 8.3 Implement protected backend-streamed download with safe Content-Disposition, context cancellation, device data permission revalidation, and GVA operation audit
- [ ] 8.4 Register list/download APIs and Casbin permissions in the TR-069 plugin initializer

## 9. GVA Log Files Page

- [ ] 9.1 Add frontend API wrappers and tests for artifact pagination, device ID filter parameters, authenticated binary download, and backend error handling
- [ ] 9.2 Add the TR-069 “日志文件” menu and route, placing it before the alarm submenu without changing existing route identities
- [ ] 9.3 Build the GVA-styled page with device ID search, reset, pagination, receive time, device identity, filename, size, checksum, source, status, and conditional download action
- [ ] 9.4 Add component tests for device ID filtering, pagination, unavailable download disabling, permission behavior, and light/dark theme token usage
- [ ] 9.5 Regenerate or update frontend route component metadata using the repository's supported generator and verify production build resolution

## 10. Integration, Security, and Operations

- [ ] 10.1 Add MinIO and file-ingress example configuration, secret injection documentation, bucket bootstrap, health checks, and Docker development wiring without changing automatic container restart policy
- [ ] 10.2 Add end-to-end Basic/Digest clients for both PUT and POST that perform Inform binding, stream a representative raw log body, verify MinIO metadata and cross-method idempotency, filter by device ID, and download matching bytes
- [ ] 10.3 Run existing TR-069 CWMP/core/backend/frontend regression suites and verify POST `/acs` behavior is unchanged
- [ ] 10.4 Test a real BS periodic upload after the user configures `Device.LogMgmt.*`, then test active Upload/TransferComplete without executing unrelated dangerous RPCs
- [ ] 10.5 Run race/static checks and verify a 64 MiB bounded upload does not create whole-file allocations or file-body log entries
- [ ] 10.6 Document the shared-credential/NAT limitation, HTTPS production requirement, failure codes, state recovery, retention, credential rotation, and Ceph/S3 migration procedure
