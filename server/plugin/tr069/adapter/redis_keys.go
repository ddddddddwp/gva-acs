package adapter

const (
	RedisIngestStreamKey    = "tr069:cmd:ingest"
	RedisDispatcherGroup    = "tr069:cmd:dispatchers"
	RedisPendingListPrefix  = "tr069:cmd:pending:"
	RedisDeviceLockPrefix   = "tr069:cmd:lock:"
	RedisInflightHashPrefix = "tr069:cmd:inflight:"
	RedisDedupPrefix        = "tr069:cmd:dedup:"
	RedisCommandWakeupKey   = "tr069:command:wakeup"
)
