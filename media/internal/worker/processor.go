package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"microservices/media/internal/config"
	"microservices/media/internal/domain"
)

type ImageProcessor struct {
	storage     domain.StorageClient
	queue       domain.MessageQueue
	cfg         config.MediaConfig
	subject     string
	workerCount int
	jobs        chan domain.MediaEvent
	wg          sync.WaitGroup
}

func NewImageProcessor(storage domain.StorageClient, queue domain.MessageQueue, cfg config.MediaConfig, subject string, workerCount int) *ImageProcessor {
	if workerCount <= 0 {
		workerCount = 4
	}
	return &ImageProcessor{
		storage:     storage,
		queue:       queue,
		cfg:         cfg,
		subject:     subject,
		workerCount: workerCount,
		jobs:        make(chan domain.MediaEvent, 100),
	}
}

func (p *ImageProcessor) Start(ctx context.Context) error {
	// Start worker pool
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Subscribe to events
	err := p.queue.Subscribe(ctx, p.subject, func(data []byte) error {
		var event domain.MediaEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to unmarshal event: %w", err)
		}

		select {
		case p.jobs <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

func (p *ImageProcessor) Stop() {
	close(p.jobs)
	p.wg.Wait()
}

func (p *ImageProcessor) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-p.jobs:
			if !ok {
				return
			}
			if err := p.processImage(ctx, event); err != nil {
				fmt.Printf("Worker %d: Error processing %s: %v\n", id, event.Key, err)
			} else {
				fmt.Printf("Worker %d: Successfully processed %s\n", id, event.Key)
			}
		}
	}
}

func (p *ImageProcessor) processImage(ctx context.Context, event domain.MediaEvent) error {
	// Check if it's an image file
	if !isImageFile(event.Key) {
		fmt.Printf("Skipping non-image file: %s\n", event.Key)
		return nil
	}

	// Download original image
	data, err := p.storage.GetObject(ctx, event.Key)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	// Detect MIME type
	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		return fmt.Errorf("file is not an image: %s", mimeType)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Create temp directory
	tempDir := filepath.Join(p.cfg.TempDir, "media-processing")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate thumbnail
	thumbnail := imaging.Fit(img, p.cfg.ThumbnailSize, p.cfg.ThumbnailSize, imaging.Lanczos)
	thumbData, err := encodeImage(thumbnail, format)
	if err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Generate medium size
	medium := imaging.Fit(img, p.cfg.MediumSize, p.cfg.MediumSize, imaging.Lanczos)
	mediumData, err := encodeImage(medium, format)
	if err != nil {
		return fmt.Errorf("failed to encode medium: %w", err)
	}

	// Generate new keys for processed images
	ext := filepath.Ext(event.Key)
	baseName := strings.TrimSuffix(event.Key, ext)
	thumbKey := baseName + "_thumb" + ext
	mediumKey := baseName + "_medium" + ext

	// Upload processed images
	contentType := "image/" + format
	if err := p.storage.PutObject(ctx, thumbKey, thumbData, contentType); err != nil {
		return fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	if err := p.storage.PutObject(ctx, mediumKey, mediumData, contentType); err != nil {
		return fmt.Errorf("failed to upload medium: %w", err)
	}

	return nil
}

func isImageFile(key string) bool {
	ext := strings.ToLower(filepath.Ext(key))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, e := range imageExts {
		if ext == e {
			return true
		}
	}
	return false
}

func encodeImage(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	var imgFormat imaging.Format
	switch format {
	case "jpeg":
		imgFormat = imaging.JPEG
	case "png":
		imgFormat = imaging.PNG
	case "gif":
		imgFormat = imaging.GIF
	default:
		imgFormat = imaging.JPEG
	}

	if err := imaging.Encode(&buf, img, imgFormat); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
