## ADDED Requirements

### Requirement: Shared LOG channel authentication
The system SHALL authenticate every `PUT /acs/log` and `POST /acs/log` request with the configured non-empty shared LOG username and password, SHALL support HTTP Basic and Digest for both methods, and SHALL fail plugin startup when the enabled channel has invalid credentials.

#### Scenario: Enabled channel has empty credentials
- **WHEN** the LOG file ingress channel is enabled and its configured username or password is empty
- **THEN** the TR-069 plugin SHALL refuse to start the file ingress and SHALL report a non-secret configuration error

#### Scenario: Request has no authentication
- **WHEN** a client sends `PUT /acs/log` or `POST /acs/log` without valid Basic or Digest authentication
- **THEN** the system SHALL return `401` with an authentication challenge and SHALL not read or persist the file body

#### Scenario: Shared credentials are valid
- **WHEN** a client sends valid configured Basic or Digest credentials with either supported upload method
- **THEN** the system SHALL authorize access to the LOG channel without treating the username as a device identifier

### Requirement: PUT and POST upload compatibility
The system SHALL accept both PUT and POST for every enabled file-ingress channel, SHALL process both methods through the same authentication, identity, admission, streaming, storage, and state pipeline, and SHALL treat each request body as the raw artifact byte stream.

#### Scenario: Device uploads with PUT
- **WHEN** an authenticated and uniquely resolved device sends `PUT /acs/log` with a raw file body
- **THEN** the system SHALL process the file through the LOG ingress pipeline

#### Scenario: Device uploads with POST
- **WHEN** an authenticated and uniquely resolved device sends `POST /acs/log` with a raw file body
- **THEN** the system SHALL process the file identically to PUT without requiring multipart form data

#### Scenario: Unsupported method is used
- **WHEN** a client uses a method other than PUT or POST on an enabled file-ingress path
- **THEN** the system SHALL return `405` with `Allow: PUT, POST` and SHALL not read or persist an artifact body

### Requirement: Unique registered-device resolution
The system SHALL resolve an authenticated upload to exactly one registered device using its trusted source IP, recent Inform identity binding, and any unique active Upload task, and SHALL reject requests that cannot be resolved uniquely.

#### Scenario: Recent Inform uniquely binds the source IP
- **WHEN** an authenticated upload arrives from a source IP that has one valid recent Inform binding to a registered device
- **THEN** the system SHALL associate the upload with that device ID and SHALL record the OUI, ProductClass, and SerialNumber identity used for verification

#### Scenario: Multiple devices match the source IP
- **WHEN** more than one registered device or active task can match the authenticated request's source IP
- **THEN** the system SHALL reject the upload with a generic `403` response and SHALL internally record `DEVICE_AMBIGUOUS`

#### Scenario: Device is unknown or binding expired
- **WHEN** no currently registered device can be uniquely resolved within the configured identity-binding window
- **THEN** the system SHALL reject the upload with a generic `403` response and SHALL not create an available artifact

#### Scenario: Forwarded address comes from an untrusted peer
- **WHEN** a request contains forwarded-IP headers but its direct peer is not in the configured trusted-proxy ranges
- **THEN** the system SHALL ignore those headers and use the direct peer address for device resolution

### Requirement: Inform refreshes upload identity binding
The system SHALL refresh a TTL-bound source-IP-to-device binding only after a valid Inform has been accepted and the device identity has been persisted.

#### Scenario: Registered device sends Inform
- **WHEN** a valid Inform for a registered or newly registered device is successfully persisted
- **THEN** the system SHALL store a binding containing device ID, OUI, ProductClass, SerialNumber, normalized source IP, and Inform time for the configured TTL

#### Scenario: Invalid Inform is rejected
- **WHEN** an Inform fails protocol validation or device persistence
- **THEN** the system SHALL NOT create or refresh a file-ingress identity binding

### Requirement: Route-specific body handling
The system SHALL process CWMP XML and file-upload routes through separate middleware chains, and SHALL never send a file body through RawDump or XML parsing.

#### Scenario: Log file is uploaded
- **WHEN** an authenticated and resolved device sends `PUT /acs/log` or `POST /acs/log`
- **THEN** the request body SHALL be streamed directly to the artifact store with bounded buffering and SHALL not be copied into XML logs, traces, or database fields

#### Scenario: CWMP message is posted
- **WHEN** a device sends `POST /acs`
- **THEN** the existing CWMP XML parsing and logging behavior SHALL remain active and SHALL not use the file-ingress handler

### Requirement: Bounded streaming upload
The system SHALL enforce configured file-size, timeout, global concurrency, channel concurrency, and per-device concurrency limits while calculating SHA-256 during streaming.

#### Scenario: File within configured limits succeeds
- **WHEN** a uniquely resolved device uploads a file within all configured limits and MinIO commit succeeds
- **THEN** the system SHALL persist its size and SHA-256, mark the artifact available, release all concurrency tokens, and return `204`

#### Scenario: Declared file is too large
- **WHEN** Content-Length exceeds the configured LOG maximum
- **THEN** the system SHALL return `413` before starting object storage and SHALL not create an available artifact

#### Scenario: Chunked file exceeds the limit
- **WHEN** a body without a usable Content-Length exceeds the configured LOG maximum while streaming
- **THEN** the system SHALL abort the object upload, mark the receive attempt failed, and return `413`

#### Scenario: Device concurrency is exhausted
- **WHEN** another LOG upload is already active for the same device
- **THEN** the system SHALL return `503` with `Retry-After` and SHALL not consume the new file body

#### Scenario: Client disconnects during upload
- **WHEN** the request context is canceled before storage commit
- **THEN** the system SHALL abort the multipart upload, release all tokens, and record a failed transfer event without retaining a downloadable partial object

### Requirement: Active and periodic transfer classification
The system SHALL associate a received file with the device's unique waiting active Upload task when one exists; otherwise it SHALL create a periodic LOG transfer task.

#### Scenario: Unique active Upload task is waiting
- **WHEN** a resolved device uploads a LOG file while exactly one active task is in `WAITING_FILE`
- **THEN** the system SHALL attach the artifact to that task and preserve its CommandKey association

#### Scenario: No active Upload task is waiting
- **WHEN** a resolved device uploads a LOG file and has no waiting active task
- **THEN** the system SHALL create a `PERIODIC` transfer task and associate the artifact with it

#### Scenario: More than one active task could match
- **WHEN** data corruption or a race leaves multiple waiting active LOG tasks for the same device
- **THEN** the system SHALL reject the upload as ambiguous and SHALL not guess a task

### Requirement: Active Upload completion semantics
The system SHALL combine UploadResponse, artifact storage, and TransferComplete idempotently according to the UploadResponse status.

#### Scenario: UploadResponse status is zero
- **WHEN** an active Upload task has status `0` and its artifact becomes available
- **THEN** the system SHALL complete the transfer task without requiring TransferComplete

#### Scenario: UploadResponse status is one
- **WHEN** an active Upload task has status `1`
- **THEN** the system SHALL complete only after both an available artifact and a successful matching TransferComplete exist, regardless of arrival order

#### Scenario: TransferComplete reports failure
- **WHEN** a matching TransferComplete reports a transfer fault
- **THEN** the system SHALL mark the task failed while retaining any already available artifact for diagnosis

#### Scenario: Response is delivered twice
- **WHEN** a duplicate UploadResponse, PUT/POST file upload, or TransferComplete is received
- **THEN** conditional state updates SHALL prevent duplicate completion and duplicate event effects

### Requirement: Channel registry reserves future transfer types
The system SHALL register file ingress by channel configuration and SHALL ship PM and MR channels disabled without implementing their business processing.

#### Scenario: Disabled future channel is requested
- **WHEN** a client uses PUT or POST on `/acs/pm` or `/acs/mr` while that channel is disabled
- **THEN** the system SHALL not invoke LOG handling or create a LOG artifact

### Requirement: GVA does not manage Device.LogMgmt parameters
The system SHALL leave `Device.LogMgmt.*` configuration to the user and SHALL not introduce TR-181 mapping or automatic parameter provisioning in this change.

#### Scenario: File ingress is enabled
- **WHEN** the administrator enables the GVA LOG ingress
- **THEN** the system SHALL expose the configured upload endpoint but SHALL NOT automatically issue GetParameterValues or SetParameterValues for `Device.LogMgmt.*`
