package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// VersionHandler serves the force-update gate config. The mobile app calls
// GET /version on every cold start and blocks usage when its installed
// version is below MinimumVersion.
type VersionHandler struct {
	minimumVersion  string
	androidStoreURL string
	iosStoreURL     string
}

func NewVersionHandler(minimumVersion, androidStoreURL, iosStoreURL string) *VersionHandler {
	return &VersionHandler{
		minimumVersion:  minimumVersion,
		androidStoreURL: androidStoreURL,
		iosStoreURL:     iosStoreURL,
	}
}

// GET /version - public, no auth. Store URLs are served from config so they
// can be set/changed after launch without shipping a new binary.
func (h *VersionHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"minimum_version":   h.minimumVersion,
		"android_store_url": h.androidStoreURL,
		"ios_store_url":     h.iosStoreURL,
	})
}
