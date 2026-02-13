package api

import (
	"net/http"
	"strconv"

	"ForecastSync/internal/repository"
	"ForecastSync/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MarketHandler 提供给前端的市场查询接口
type MarketHandler struct {
	marketService *service.MarketService
	logger        *logrus.Logger
}

// NewMarketHandler 创建 MarketHandler
func NewMarketHandler(db *gorm.DB, logger *logrus.Logger) *MarketHandler {
	repo := repository.NewMarketRepository(db)
	canonicalRepo := repository.NewCanonicalRepository(db)
	svc := service.NewMarketService(repo, canonicalRepo, logger)
	return &MarketHandler{
		marketService: svc,
		logger:        logger,
	}
}

// ListMarkets 市场列表接口
// GET /api/markets?type=sports&status=active&platform=polymarket&page=1&page_size=20
func (h *MarketHandler) ListMarkets(c *gin.Context) {
	eventType := c.DefaultQuery("type", "sports")
	status := c.DefaultQuery("status", "active")
	platform := c.Query("platform")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.MarketFilter{
		Type:     eventType,
		Status:   status,
		Platform: platform,
	}

	result, err := h.marketService.ListMarkets(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("ListMarkets failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetMarketDetail 市场详情 + 平台对比。:id 为数字时即 canonical_id，否则按 event_uuid 解析所属聚合赛事
// GET /api/markets/:id
func (h *MarketHandler) GetMarketDetail(c *gin.Context) {
	idOrUUID := c.Param("event_uuid")
	if idOrUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id or event_uuid is required"})
		return
	}

	result, err := h.marketService.GetMarketDetail(c.Request.Context(), idOrUUID)
	if err != nil {
		h.logger.WithError(err).Error("GetMarketDetail failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
