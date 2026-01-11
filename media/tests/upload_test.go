package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tafu-media/media-service/internal/config"
	"github.com/tafu-media/media-service/internal/domain"
	"github.com/tafu-media/media-service/internal/handler"
	"github.com/tafu-media/media-service/internal/infrastructure/queue"
	"github.com/tafu-media/media-service/internal/infrastructure/storage"
	"github.com/tafu-media/media-service/internal/usecase"
	"github.com/tafu-media/media-service/internal/worker"
)

// TestIntegration_FullUploadFlow tests the complete upload flow:
// 1. Get upload URL
// 2. Upload file to MinIO via presigned URL
// 3. Confirm upload
// 4. Verify thumbnail creation
func TestIntegration_FullUploadFlow(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	ctx := context.Background()

	// Load config
	cfg := config.Load()

	// Initialize MinIO client
	minioClient, err := storage.NewMinIOClient(cfg.MinIO)
	if err != nil {
		t.Fatalf("Failed to create MinIO client: %v", err)
	}

	if err := minioClient.EnsureBucket(ctx); err != nil {
		t.Fatalf("Failed to ensure bucket: %v", err)
	}

	// Initialize NATS client
	natsClient, err := queue.NewNATSClient(cfg.NATS)
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer natsClient.Close()

	// Initialize use case
	mediaUseCase := usecase.NewMediaUseCase(minioClient, natsClient, cfg.Media, cfg.NATS.Subject)

	// Start image processor
	processor := worker.NewImageProcessor(minioClient, natsClient, cfg.Media, cfg.NATS.Subject, 2)
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	if err := processor.Start(workerCtx); err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Create HTTP handler and server
	mediaHandler := handler.NewMediaHandler(mediaUseCase)
	router := handler.NewRouter(mediaHandler)
	server := httptest.NewServer(router)
	defer server.Close()

	// ==========================================
	// STEP 1: Get Upload URL
	// ==========================================
	t.Log("Step 1: Getting upload URL...")

	uploadReq := map[string]string{
		"user_id":   "test-user-123",
		"file_type": "image/png",
		"purpose":   "product_image",
	}
	reqBody, _ := json.Marshal(uploadReq)

	resp, err := http.Post(server.URL+"/media/upload-url", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to get upload URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var uploadURLResponse domain.UploadURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadURLResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Got upload URL for key: %s", uploadURLResponse.FileKey)

	// ==========================================
	// STEP 2: Upload dummy image via presigned URL
	// ==========================================
	t.Log("Step 2: Uploading dummy image...")

	dummyImage := createDummyPNG(100, 100)

	putReq, err := http.NewRequest(http.MethodPut, uploadURLResponse.UploadURL, bytes.NewReader(dummyImage))
	if err != nil {
		t.Fatalf("Failed to create PUT request: %v", err)
	}
	putReq.Header.Set("Content-Type", "image/png")

	client := &http.Client{Timeout: 30 * time.Second}
	putResp, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("Failed to upload image: %v", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("Expected status 200 for PUT, got %d: %s", putResp.StatusCode, string(body))
	}

	t.Log("Image uploaded successfully")

	// ==========================================
	// STEP 3: Confirm Upload
	// ==========================================
	t.Log("Step 3: Confirming upload...")

	confirmReq := map[string]string{
		"file_key": uploadURLResponse.FileKey,
	}
	confirmBody, _ := json.Marshal(confirmReq)

	confirmResp, err := http.Post(server.URL+"/media/confirm", "application/json", bytes.NewReader(confirmBody))
	if err != nil {
		t.Fatalf("Failed to confirm upload: %v", err)
	}
	defer confirmResp.Body.Close()

	if confirmResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(confirmResp.Body)
		t.Fatalf("Expected status 200 for confirm, got %d: %s", confirmResp.StatusCode, string(body))
	}

	t.Log("Upload confirmed")

	// ==========================================
	// STEP 4: Wait and verify thumbnail exists
	// ==========================================
	t.Log("Step 4: Waiting for thumbnail generation...")

	// Wait for worker to process
	time.Sleep(3 * time.Second)

	// Check if thumbnail exists
	thumbKey := strings.TrimSuffix(uploadURLResponse.FileKey, ".png") + "_thumb.png"
	exists, err := minioClient.ObjectExists(ctx, thumbKey)
	if err != nil {
		t.Fatalf("Failed to check thumbnail existence: %v", err)
	}

	if !exists {
		t.Errorf("Thumbnail not found: %s", thumbKey)
	} else {
		t.Logf("Thumbnail created successfully: %s", thumbKey)
	}

	// Check if medium size exists
	mediumKey := strings.TrimSuffix(uploadURLResponse.FileKey, ".png") + "_medium.png"
	exists, err = minioClient.ObjectExists(ctx, mediumKey)
	if err != nil {
		t.Fatalf("Failed to check medium existence: %v", err)
	}

	if !exists {
		t.Errorf("Medium size not found: %s", mediumKey)
	} else {
		t.Logf("Medium size created successfully: %s", mediumKey)
	}

	t.Log("Integration test passed!")
}

func TestGetUploadURL_InvalidFileType(t *testing.T) {
	cfg := config.Load()
	cfg.Media.AllowedMimeTypes = []string{"image/jpeg", "image/png"}

	// Create mock storage and queue
	mockStorage := &mockStorageClient{}
	mockQueue := &mockMessageQueue{}

	mediaUseCase := usecase.NewMediaUseCase(mockStorage, mockQueue, cfg.Media, cfg.NATS.Subject)
	mediaHandler := handler.NewMediaHandler(mediaUseCase)
	router := handler.NewRouter(mediaHandler)
	server := httptest.NewServer(router)
	defer server.Close()

	// Try to upload executable file
	uploadReq := map[string]string{
		"user_id":   "test-user",
		"file_type": "application/x-executable",
		"purpose":   "product_image",
	}
	reqBody, _ := json.Marshal(uploadReq)

	resp, err := http.Post(server.URL+"/media/upload-url", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file type, got %d", resp.StatusCode)
	}
}

func TestHealthCheck(t *testing.T) {
	cfg := config.Load()
	mockStorage := &mockStorageClient{}
	mockQueue := &mockMessageQueue{}

	mediaUseCase := usecase.NewMediaUseCase(mockStorage, mockQueue, cfg.Media, cfg.NATS.Subject)
	mediaHandler := handler.NewMediaHandler(mediaUseCase)
	router := handler.NewRouter(mediaHandler)
	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// createDummyPNG creates a dummy PNG image for testing
func createDummyPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// Mock implementations for unit tests
type mockStorageClient struct{}

func (m *mockStorageClient) GeneratePresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	return "http://localhost:9000/test-bucket/" + objectName + "?presigned=true", nil
}

func (m *mockStorageClient) GetPublicURL(objectName string) string {
	return "http://localhost:9000/test-bucket/" + objectName
}

func (m *mockStorageClient) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	return true, nil
}

func (m *mockStorageClient) GetObject(ctx context.Context, objectName string) ([]byte, error) {
	return createDummyPNG(100, 100), nil
}

func (m *mockStorageClient) PutObject(ctx context.Context, objectName string, data []byte, contentType string) error {
	return nil
}

func (m *mockStorageClient) DeleteObject(ctx context.Context, objectName string) error {
	return nil
}

func (m *mockStorageClient) EnsureBucket(ctx context.Context) error {
	return nil
}

type mockMessageQueue struct{}

func (m *mockMessageQueue) Publish(ctx context.Context, subject string, data []byte) error {
	return nil
}

func (m *mockMessageQueue) Subscribe(ctx context.Context, subject string, handler func(data []byte) error) error {
	return nil
}

func (m *mockMessageQueue) Close() error {
	return nil
}
