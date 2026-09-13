module microservices/realtime

go 1.22

require (
	github.com/gofiber/contrib/websocket v1.3.2
	github.com/gofiber/fiber/v2 v2.52.5
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.38.0
	microservices/pkg/logger v0.0.0
	microservices/pkg/realtime v0.0.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/fasthttp/websocket v1.5.8 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/savsgio/gotils v0.0.0-20240303185622-093b76447511 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.52.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
)

replace microservices/pkg/logger => ../pkg/logger

replace microservices/pkg/realtime => ../pkg/realtime
