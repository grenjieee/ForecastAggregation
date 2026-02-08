package api

import (
	"ForecastSync/internal/config"
	"fmt"
	"net/http"

	"ForecastSync/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SyncHandler struct {
	syncService *service.SyncService
	logger      *logrus.Logger
}

func NewSyncHandler(db *gorm.DB, logger *logrus.Logger, cfg *config.Config) *SyncHandler {
	return &SyncHandler{
		syncService: service.NewSyncService(db, logger, cfg),
		logger:      logger,
	}
}

// SyncPlatformHandler 同步指定平台数据
// @Summary 同步平台预测数据
// @Param platform path string true "平台名称（Polymarket/Kalshi）"
// @Param type query string false "事件类型（默认sports）"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /sync/platform/{platform} [post]
func (h *SyncHandler) SyncPlatformHandler(c *gin.Context) {
	platformName := c.Param("platform")
	eventType := c.DefaultQuery("type", "sports")

	if err := h.syncService.SyncPlatform(c.Request.Context(), platformName, eventType); err != nil {
		h.logger.Errorf("同步%s失败: %v", platformName, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s同步成功", platformName),
	})
}
