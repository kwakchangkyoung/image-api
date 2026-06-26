package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"image_go_api/internal/config"
	"image_go_api/internal/util"

	"github.com/gin-gonic/gin"
)

type ImageHandler struct {
	cfg *config.Config
}

func NewImageHandler(cfg *config.Config) *ImageHandler {
	return &ImageHandler{cfg: cfg}
}

func (h *ImageHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "업로드 파일이 없습니다.",
		})
		return
	}
	defer file.Close()

	if header.Size > h.cfg.MaxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "파일 크기가 제한을 초과했습니다.",
		})
		return
	}

	reader := io.LimitReader(file, h.cfg.MaxFileSize+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "파일 읽기에 실패했습니다.",
		})
		return
	}

	if int64(len(data)) > h.cfg.MaxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "파일 크기가 제한을 초과했습니다.",
		})
		return
	}

	mimeType := http.DetectContentType(data)
	ext, ok := h.cfg.AllowedMIME[mimeType]
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": "지원하지 않는 이미지 형식입니다.",
		})
		return
	}

	filename := util.GenerateFilename(ext)
	subdir := filename[:2]
	relPath := filepath.ToSlash(filepath.Join(subdir, filename))
	dirPath := filepath.Join(h.cfg.UploadDir, subdir)
	fullPath := filepath.Join(dirPath, filename)

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "저장 디렉토리 생성에 실패했습니다.",
		})
		return
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "파일 저장에 실패했습니다.",
		})
		return
	}

	baseURL := strings.TrimRight(h.cfg.BaseURL, "/")

	c.JSON(http.StatusOK, gin.H{
		"filename":  filename,
		"path":      relPath,
		"url":       baseURL + "/api/images/" + relPath,
		"size":      len(data),
		"mime_type": mimeType,
		"md5":       util.MD5Hex(data),
	})
}

func (h *ImageHandler) Get(c *gin.Context) {
	subdir := c.Param("subdir")
	filename := c.Param("filename")

	filePath, ok := util.SafePath(h.cfg.UploadDir, subdir, filename)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "잘못된 경로입니다.",
		})
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "파일을 찾을 수 없습니다.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "파일 조회에 실패했습니다.",
		})
		return
	}

	c.Header("Cache-Control", "public, max-age=2592000, immutable")
	c.File(filePath)
}

func (h *ImageHandler) Delete(c *gin.Context) {
	subdir := c.Param("subdir")
	filename := c.Param("filename")

	filePath, ok := util.SafePath(h.cfg.UploadDir, subdir, filename)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "잘못된 경로입니다.",
		})
		return
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "파일을 찾을 수 없습니다.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "파일 삭제에 실패했습니다.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "삭제되었습니다.",
		"filename": filename,
	})
}

func (h *ImageHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"upload_dir": h.cfg.UploadDir,
	})
}
