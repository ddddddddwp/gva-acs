# Durable Dispatch Second Review Design

## Goal

Close the remaining command-dispatch races without changing the required `CommandRepo` interface or introducing process-local synchronization that fails across ACS instances.

## State arbitration

The command row and its optimistic `version` remain authoritative. Wake enqueue compensation locks and re-reads the row. It may transition only `WAITING_DEVICE` or `BUILDING` to `FAILED`. If the row is already `SENT`, `WAITING_TRANSFER`, or terminal, the wake token is redundant: return the actual durable status without an error or a false failure transition.

Core orders a candidate request as correlation hook, durable `MarkSending`, ownership ack, then outbound return. This creates a deterministic race boundary. If wake compensation wins while the row is `BUILDING`, the later `MarkSending` conflicts and core returns no XML. If `MarkSending` wins, compensation observes `SENT` and returns that status unchanged.

## Core failure classification

Correlation failure occurs before `REQUEST_SENT`. Core marks the unsent command terminal; if persistence fails, it nacks to restore recoverability and aggregates persistence/nack errors while returning no outbound message.

Ack failure occurs after `MarkSending`. Core exposes an optional `StagedCommandRepo` interface for repositories that can persist an explicit failure stage. Existing `CommandRepo` implementations remain compatible through the original `MarkFail` fallback. The GVA repository implements the optional interface and records ownership-ack failures as `queue.ack`, never `cwmp.fault`.

## Shutdown lifecycle

Shutdown has two independent time budgets. First, HTTP `Shutdown` stops acceptance and drains active requests while background workers remain available. Second, a fresh cleanup context runs plugin shutdown handlers. Handler registration and triggering tolerate nil receivers, nil handlers, and nil contexts. Each handler runs behind panic recovery; returned errors and recovered panics are joined so later handlers always execute.

## Connection Request result semantics

Only HTTP 2xx is a successful Connection Request. Every non-2xx response becomes an error carrying its status and participates in the configured bounded retry loop. An exhausted non-2xx sequence returns an error to the wake consumer, which records `WAKE_FAILED`; a 2xx response records `WAKE_TRIGGERED`.

Integration tests use a real TLS test server and an injected HTTP client/transport seam so production HTTPS validation remains intact without globally disabling certificate verification.

## Verification

Tests cover both exact wake/core interleavings, hook persistence and recovery failures, optional staged-repository compatibility, shutdown ordering and isolation, real 2xx/401/500 Connection Request behavior, API result envelopes, race detection, repeated critical tests, all TR-069 parent packages, and the complete standalone-core suite.
