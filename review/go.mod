module microservices/review

go 1.22

require (
	github.com/go-playground/validator/v10 v10.22.0
	github.com/gofiber/fiber/v2 v2.52.5
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.37.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/spf13/viper v1.18.2
	go.mongodb.org/mongo-driver v1.17.1
	google.golang.org/grpc v1.68.1
	google.golang.org/protobuf v1.35.2
	microservices/pkg/authclient v0.0.0
	microservices/pkg/authorization v0.0.0
	microservices/pkg/cache v0.0.0
	microservices/pkg/customfields v0.0.0
	microservices/pkg/logger v0.0.0
	microservices/pkg/response v0.0.0
)

replace (
	microservices/pkg/authclient => ../pkg/authclient
	microservices/pkg/authorization => ../pkg/authorization
	microservices/pkg/cache => ../pkg/cache
	microservices/pkg/customfields => ../pkg/customfields
	microservices/pkg/logger => ../pkg/logger
	microservices/pkg/response => ../pkg/response
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
)
