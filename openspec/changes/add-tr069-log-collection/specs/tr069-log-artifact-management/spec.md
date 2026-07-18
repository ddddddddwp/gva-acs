## ADDED Requirements

### Requirement: Independent transfer metadata
The system SHALL persist TR-069 transfer tasks, artifacts, and transfer events in plugin-owned tables rather than the GVA generic attachment table.

#### Scenario: Upload receive begins
- **WHEN** an authenticated upload has been uniquely resolved and admitted
- **THEN** the system SHALL create auditable task/artifact metadata in `RECEIVING` state before making a file downloadable

#### Scenario: Upload fails
- **WHEN** storage, checksum, size, timeout, or client transport fails
- **THEN** the system SHALL retain a failure event and terminal or recoverable state without exposing a partial file for download

### Requirement: Storage-provider abstraction
The system SHALL write and read artifact bytes through a streaming store abstraction whose production implementation supports MinIO/S3-compatible object storage.

#### Scenario: MinIO upload succeeds
- **WHEN** the file stream is committed successfully to the configured MinIO bucket and database metadata is updated
- **THEN** the artifact SHALL become `AVAILABLE` with an opaque service-generated object key, size, and SHA-256

#### Scenario: Object storage driver changes
- **WHEN** a future deployment selects another S3-compatible service such as Ceph RGW
- **THEN** transfer authentication, task state, management APIs, and the GVA page SHALL not require provider-specific changes

### Requirement: Cross-store reconciliation
The system SHALL reconcile incomplete MySQL/object-storage operations and SHALL expose only artifacts confirmed available in both metadata and storage.

#### Scenario: Object commit succeeds but metadata finalization is interrupted
- **WHEN** a stale `RECEIVING` record has a matching object with expected metadata
- **THEN** the reconciler SHALL idempotently finalize the artifact as `AVAILABLE`

#### Scenario: Stale receive has no valid object
- **WHEN** a `RECEIVING` record exceeds its timeout and no matching complete object exists
- **THEN** the reconciler SHALL abort or delete residual storage and mark the artifact failed

### Requirement: Log artifact query API
The system SHALL provide a JWT/Casbin-protected paginated management API that supports exact filtering by device ID.

#### Scenario: User filters by device ID
- **WHEN** an authorized user requests the artifact list with a device ID
- **THEN** the system SHALL return only LOG artifacts belonging to that device and permitted by the user's device data scope

#### Scenario: User does not provide device ID
- **WHEN** an authorized user requests the artifact list without a device ID
- **THEN** the system SHALL return a paginated list limited to devices within that user's data scope

#### Scenario: Unauthorized list access
- **WHEN** a user lacks the artifact-list permission
- **THEN** the system SHALL deny the request without revealing artifact metadata

### Requirement: Protected artifact download
The system SHALL stream an available artifact to an authorized GVA user only after revalidating JWT, Casbin, device data permission, and artifact state.

#### Scenario: Authorized user downloads available artifact
- **WHEN** a user with access to the artifact's device requests its download and the artifact is `AVAILABLE`
- **THEN** the backend SHALL stream the object with a safe filename and SHALL record a GVA operation audit entry

#### Scenario: User lacks device access
- **WHEN** a user can call the endpoint but lacks data permission for the artifact's device
- **THEN** the system SHALL deny download without exposing the MinIO object key or storage credentials

#### Scenario: Artifact is not available
- **WHEN** an artifact is receiving, failed, deleting, deleted, or missing in object storage
- **THEN** the system SHALL not return a file body and SHALL return an appropriate non-success response

### Requirement: TR-069 log files menu
The system SHALL add a “日志文件” page under the TR-069 menu that follows the active GVA light/dark theme and exposes device-filtered query and download operations.

#### Scenario: User opens the page
- **WHEN** an authorized user opens the TR-069 “日志文件” menu
- **THEN** the page SHALL show a GVA-styled search area, paginated table, device ID filter, artifact metadata, status, and permitted download action

#### Scenario: User changes GVA theme
- **WHEN** the application switches between supported light and dark themes
- **THEN** the page SHALL use GVA/Element Plus theme tokens without fixed incompatible colors

#### Scenario: User downloads from the table
- **WHEN** an authorized user clicks download on an available row
- **THEN** the frontend SHALL call the protected management API and SHALL not construct a MinIO URL from row data

### Requirement: Artifact metadata confidentiality
The system SHALL not expose shared upload credentials, storage credentials, internal filesystem paths, or raw object keys in list/download API responses or routine logs.

#### Scenario: Artifact list is returned
- **WHEN** the management API serializes an artifact row
- **THEN** it SHALL include business metadata such as device ID, identity, filename, size, checksum, source, status, and receive time but SHALL omit secret and provider-internal fields

### Requirement: Retention cleanup
The system SHALL apply the configured LOG retention period with idempotent delete states while preserving task/event audit metadata.

#### Scenario: Available artifact reaches retention deadline
- **WHEN** an artifact is older than the configured retention period
- **THEN** the cleanup worker SHALL mark it deleting, remove its stored bytes, mark it deleted, and preserve its task and non-secret event history

#### Scenario: Storage deletion fails
- **WHEN** object deletion fails after the artifact enters deleting state
- **THEN** the system SHALL record the failure and retry without reporting the artifact as deleted
