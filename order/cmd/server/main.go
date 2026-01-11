package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/tafu/order-service/internal/config"
	"github.com/tafu/order-service/internal/handler/http"
	"github.com/tafu/order-service/internal/infrastructure/database"
	inventorygrpc "github.com/tafu/order-service/internal/infrastructure/grpc"
	"github.com/tafu/order-service/internal/infrastructure/messaging"
	"github.com/tafu/order-service/internal/infrastructure/repository"
	"github.com/tafu/order-service/internal/usecase"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize inventory gRPC client
	inventoryClient, err := inventorygrpc.NewInventoryClient(&cfg.Inventory)
	if err != nil {
		log.Printf("Warning: Failed to connect to inventory service: %v", err)
		// Continue without inventory service for development
	} else {
		defer inventoryClient.Close()
	}

	// Initialize NATS publisher
	natsPublisher, err := messaging.NewNATSPublisher(&cfg.NATS)
	if err != nil {
		log.Printf("Warning: Failed to connect to NATS: %v", err)
		// Continue without NATS for development
	} else {
		defer natsPublisher.Close()
	}

	// Initialize repository
	orderRepo := repository.NewOrderRepository(db)

	// Create mock implementations for development if services are unavailable
	var stockReserver inventorygrpc.StockReserver = inventoryClient
	if inventoryClient == nil {
		stockReserver = &mockStockReserver{}
		log.Println("Using mock stock reserver")
	}

	var eventPublisher messaging.EventPublisher = natsPublisher
	if natsPublisher == nil {
		eventPublisher = &mockEventPublisher{}
		log.Println("Using mock event publisher")
	}

	// Initialize use cases
	createOrderUC := usecase.NewCreateOrderUseCase(orderRepo, stockReserver, eventPublisher)
	cancelOrderUC := usecase.NewCancelOrderUseCase(orderRepo, stockReserver, eventPublisher)
	getOrderUC := usecase.NewGetOrderUseCase(orderRepo)
	listOrdersUC := usecase.NewListOrdersUseCase(orderRepo)
	markAsPaidUC := usecase.NewMarkAsPaidUseCase(orderRepo, eventPublisher)
	markAsShippedUC := usecase.NewMarkAsShippedUseCase(orderRepo, eventPublisher)
	markAsCompletedUC := usecase.NewMarkAsCompletedUseCase(orderRepo, eventPublisher)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Initialize handlers
	healthHandler := http.NewHealthHandler(db)
	orderHandler := http.NewOrderHandler(
		createOrderUC,
		cancelOrderUC,
		getOrderUC,
		listOrdersUC,
		markAsPaidUC,
		markAsShippedUC,
		markAsCompletedUC,
	)

	// Register routes
	healthHandler.RegisterRoutes(app)
	orderHandler.RegisterRoutes(app)

	// Initialize structured logger
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create stock confirmer for payment listener
	var stockConfirmer inventorygrpc.StockConfirmer = inventoryClient
	if inventoryClient == nil {
		stockConfirmer = &mockStockConfirmer{}
		log.Println("Using mock stock confirmer for payment listener")
	}

	// Initialize payment event listener
	paymentListener, err := messaging.NewPaymentEventListener(
		&cfg.NATS,
		orderRepo,
		stockConfirmer,
		eventPublisher,
		slogger,
	)
	if err != nil {
		log.Printf("Warning: Failed to create payment listener: %v", err)
	}

	// Start payment listener in background
	listenerCtx, listenerCancel := context.WithCancel(context.Background())
	if paymentListener != nil {
		go func() {
			if err := paymentListener.Start(listenerCtx); err != nil {
				log.Printf("Payment listener error: %v", err)
			}
		}()
		log.Println("Payment event listener started")
	}

	// Start server
	go func() {
		addr := cfg.Server.Host + ":" + cfg.Server.Port
		log.Printf("Starting server on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop payment listener
	listenerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"message": err.Error(),
	})
}

// Mock implementations for development
type mockStockReserver struct{}

func (m *mockStockReserver) ReserveStock(ctx context.Context, skuID string, quantity int, orderID string) (string, error) {
	log.Printf("[MOCK] Reserving stock: SKU=%s, Qty=%d, OrderID=%s", skuID, quantity, orderID)
	return "mock-reservation-" + skuID, nil
}

func (m *mockStockReserver) ReleaseStock(ctx context.Context, reservationID string) error {
	log.Printf("[MOCK] Releasing stock: ReservationID=%s", reservationID)
	return nil
}

type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishOrderCreated(ctx context.Context, event *messaging.OrderCreatedEvent) error {
	log.Printf("[MOCK] Publishing order.created: OrderID=%s", event.OrderID)
	return nil
}

func (m *mockEventPublisher) PublishOrderCancelled(ctx context.Context, event *messaging.OrderCancelledEvent) error {
	log.Printf("[MOCK] Publishing order.cancelled: OrderID=%s", event.OrderID)
	return nil
}

func (m *mockEventPublisher) PublishOrderPaid(ctx context.Context, event *messaging.OrderPaidEvent) error {
	log.Printf("[MOCK] Publishing order.paid: OrderID=%s", event.OrderID)
	return nil
}

func (m *mockEventPublisher) PublishOrderShipped(ctx context.Context, event *messaging.OrderShippedEvent) error {
	log.Printf("[MOCK] Publishing order.shipped: OrderID=%s", event.OrderID)
	return nil
}

func (m *mockEventPublisher) PublishOrderCompleted(ctx context.Context, event *messaging.OrderCompletedEvent) error {
	log.Printf("[MOCK] Publishing order.completed: OrderID=%s", event.OrderID)
	return nil
}

// Mock stock confirmer for development
type mockStockConfirmer struct{}

func (m *mockStockConfirmer) ConfirmStock(ctx context.Context, orderID string) error {
	log.Printf("[MOCK] Confirming stock for order: %s", orderID)
	return nil
}

func (m *mockStockConfirmer) ReleaseStockByOrderID(ctx context.Context, orderID string) error {
	log.Printf("[MOCK] Releasing stock for order: %s", orderID)
	return nil
}
