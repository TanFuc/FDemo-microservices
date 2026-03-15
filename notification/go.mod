module microservices/notification

go 1.22

require (
	github.com/aymerick/raymond v2.0.2+incompatible
	github.com/gofiber/contrib/websocket v1.3.0
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/google/uuid v1.5.0
	github.com/nats-io/nats.go v1.31.0
	github.com/rabbitmq/amqp091-go v1.9.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/spf13/viper v1.18.2
	go.mongodb.org/mongo-driver v1.13.1
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	microservices/pkg/authorization v0.0.0
	microservices/pkg/authclient v0.0.0
	microservices/pkg/cache v0.0.0
	microservices/pkg/logger v0.0.0
	microservices/pkg/response v0.0.0
)

replace (
	microservices/pkg/authorization => ../pkg/authorization
	microservices/pkg/authclient => ../pkg/authclient
	microservices/pkg/cache => ../pkg/cache
	microservices/pkg/logger => ../pkg/logger
	microservices/pkg/response => ../pkg/response
)

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/nats-io/nkeys v0.4.6 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
)

require microservices/pkg/safego v0.0.0

replace microservices/pkg/safego => ../pkg/safego
