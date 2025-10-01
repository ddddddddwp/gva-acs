module tr069-debug-ui

go 1.24

require (
	github.com/gorilla/mux v1.8.1
	github.com/root/demo/tr069 v0.0.0-00010101000000-000000000000
	github.com/rs/cors v1.11.1
)

require github.com/google/uuid v1.6.0 // indirect

replace github.com/root/demo/tr069 => ../
