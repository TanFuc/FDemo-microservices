# MISSION: BUILD HIGH-PERFORMANCE MEDIA SERVICE (GO + MINIO + NATS)

**Role:** Infrastructure Engineer & Golang Expert.
**Goal:** Build a `media-service` that handles secure file uploads using **Presigned URLs** (Direct-to-S3 pattern) and processes images asynchronously (Resizing/Thumbnailing).

**Tech Stack:**
- **Language:** Go 1.22+.
- **Object Storage:** MinIO (S3 Compatible API) - using `github.com/minio/minio-go/v7`.
- **Image Processing:** `github.com/disintegration/imaging`.
- **Queue:** NATS JetStream (To trigger background processing).
- **Architecture:** Clean Architecture.

---

## PHASE 1: INFRASTRUCTURE (MINIO & NATS)

### 1. MinIO Setup
- **Bucket:** `ecommerce-media`.
- **Policy:** Public Read (for serving images), Private Write.
- **Lifecycle:** Auto-delete temp files older than 24h.

### 2. NATS Setup
- Stream: `MEDIA`.
- Subject: `media.uploaded`.

## PHASE 2: PRESIGNED URL LOGIC (THE UPLOAD STRATEGY)

We DO NOT upload files through the Go API Server. We authorize the client to upload directly to MinIO.

### 1. `GetUploadURL(ctx, userId, fileType)`
- **Input:** `fileType` (e.g., "image/jpeg", "video/mp4"), `purpose` (e.g., "product_image", "avatar").
- **Logic:**
    - Generate unique object name: `products/{userId}/{uuid}.jpg`.
    - Call MinIO `PresignedPutObject(bucket, name, expiry=15min)`.
- **Output:**
    - `uploadUrl`: The URL for Frontend to PUT the file.
    - `fileKey`: The key to save in DB later.
    - `publicUrl`: The final URL to view the image.

## PHASE 3: ASYNC PROCESSING WORKER

After the Frontend uploads successfully, it notifies the Backend, or we rely on MinIO Bucket Notifications (but usually Frontend trigger is easier to implement for hybrid apps). Let's use an API trigger `ConfirmUpload`.

### 1. API: `POST /media/confirm`
- **Input:** `fileKey`.
- **Logic:**
    - Validate file exists in MinIO.
    - Publish Event to NATS: `media.uploaded` -> Payload `{ key: "..." }`.
    - Return Success.

### 2. Worker: `ImageProcessor`
- **Subscribe:** `media.uploaded`.
- **Logic:**
    - Download file from MinIO to Temp.
    - Check MIME type.
    - **If Image:**
        - Generate Thumbnail (200x200).
        - Generate Medium (800x800).
        - Upload back to MinIO (`{original_key}_thumb.jpg`).
    - **Cleanup:** Delete temp files.

## PHASE 4: AUTO-VERIFICATION LOOP

**Instructions for AI:**
1.  **Init:** Setup Go project.
2.  **Generate:**
    - `internal/infrastructure/storage`: MinIO Client wrapper.
    - `internal/usecase`: Presigned URL generation.
    - `internal/worker`: Image resizing logic.
3.  **TEST (Integration):**
    - Create `tests/upload_test.go`.
    - **Step 1:** Call `GetUploadURL`. Get the PUT URL.
    - **Step 2:** Use Go `http.Client` to PUT a dummy image to that URL (Simulate Frontend).
    - **Step 3:** Call `ConfirmUpload`.
    - **Step 4:** Wait 2s. Check MinIO if `{key}_thumb.jpg` exists.
4.  **Build:** Run `go build`.

---

## IMPORTANT RULES
- **Security:** Validate `fileType` strictly in `GetUploadURL`. Don't allow `.exe` or `.sh` files.
- **Naming:** Use UUIDs for filenames to prevent collisions and encoding issues.
- **Performance:** The Worker must process images in parallel (Worker Pool pattern) to avoid backlog.