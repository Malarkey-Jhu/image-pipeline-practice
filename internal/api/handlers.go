package api

import (
	"context"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"sys-design/internal/db"
	"sys-design/internal/mq"
	"sys-design/internal/obs"
	"sys-design/internal/storage"
)

type Server struct {
	DB        *pgxpool.Pool
	Store     *storage.MinioStore
	Publisher *mq.Publisher
}

type UploadURLRequest struct {
	ContentType string `json:"content_type"`
	FileName    string `json:"file_name"`
}

type UploadURLResponse struct {
	MediaID     string `json:"media_id"`
	UploadURL   string `json:"upload_url"`
	OriginalKey string `json:"original_key"`
	ExpiresIn   int    `json:"expires_in"`
}

type CompleteUploadRequest struct {
	MediaID     string `json:"media_id"`
	OriginalKey string `json:"original_key"`
}

type MediaResponse struct {
	MediaID  string `json:"media_id"`
	Status   string `json:"status"`
	FinalURL string `json:"final_url,omitempty"`
}

func (s *Server) RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/metrics", gin.WrapH(obs.MetricsHandler()))
	r.POST("/upload-url", s.handleUploadURL)
	r.POST("/complete-upload", s.handleCompleteUpload)
	r.GET("/media/:id", s.handleGetMedia)
}

func (s *Server) handleUploadURL(c *gin.Context) {
	var req UploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	if req.ContentType != "" && !strings.HasPrefix(req.ContentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_type must be image/*"})
		return
	}

	mediaID := ulid.Make().String()
	ext := path.Ext(req.FileName)
	if ext == "" {
		ext = ".bin"
	}

	originalKey := "media/" + mediaID + "/original" + ext
	expiry := 5 * time.Minute

	uploadURL, err := s.Store.PresignUpload(context.Background(), originalKey, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to presign"})
		return
	}

	if err := db.InsertMedia(context.Background(), s.DB, mediaID, "INIT", originalKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create media"})
		return
	}
	obs.TasksCreated.Inc()

	c.JSON(http.StatusOK, UploadURLResponse{
		MediaID:     mediaID,
		UploadURL:   uploadURL,
		OriginalKey: originalKey,
		ExpiresIn:   int(expiry.Seconds()),
	})
}

func (s *Server) handleCompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if req.MediaID == "" || req.OriginalKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media_id and original_key required"})
		return
	}

	if err := db.UpdateMediaStatus(context.Background(), s.DB, req.MediaID, "PROCESSING"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update media"})
		return
	}

	// Create first task (resize) with deterministic output key.
	taskID := ulid.Make().String()
	outputKey := "media/" + req.MediaID + "/resize.jpg"
	inserted, err := db.InsertProcessingTask(context.Background(), s.DB, db.ProcessingTaskInput{
		ID:        taskID,
		MediaID:   req.MediaID,
		Step:      "resize",
		Status:    "PENDING",
		InputKey:  req.OriginalKey,
		OutputKey: outputKey,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	if inserted && s.Publisher != nil {
		_ = s.Publisher.PublishTask(mq.TaskMessage{
			TaskID:  taskID,
			MediaID: req.MediaID,
			Step:    "resize",
		})
		obs.TasksPublished.Inc()
	}

	c.JSON(http.StatusOK, gin.H{"status": "PROCESSING"})
}

func (s *Server) handleGetMedia(c *gin.Context) {
	id := c.Param("id")
	m, err := db.GetMedia(context.Background(), s.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	resp := MediaResponse{
		MediaID: m.ID,
		Status:  m.Status,
	}
	if m.FinalKey != nil {
		resp.FinalURL = *m.FinalKey
	}

	c.JSON(http.StatusOK, resp)
}
