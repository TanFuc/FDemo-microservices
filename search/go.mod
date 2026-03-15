module microservices/search

go 1.22

require (
	github.com/elastic/go-elasticsearch/v8 v8.15.0
	github.com/gofiber/fiber/v2 v2.52.5
	github.com/nats-io/nats.go v1.37.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.9.0
	microservices/pkg/cache v0.0.0
	microservices/pkg/customfields v0.0.0
	microservices/pkg/logger v0.0.0
	microservices/pkg/response v0.0.0
)

replace (
	microservices/pkg/cache => ../pkg/cache
	microservices/pkg/customfields => ../pkg/customfields
	microservices/pkg/logger => ../pkg/logger
	microservices/pkg/response => ../pkg/response
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/elastic/elastic-transport-go/v8 v8.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.55.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require microservices/pkg/safego v0.0.0

replace microservices/pkg/safego => ../pkg/safego
