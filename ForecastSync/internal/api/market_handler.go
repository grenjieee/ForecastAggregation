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

// ListMarkets 市场列表接口（一期仅 Sports）
// GET /api/markets?status=active&page=1&page_size=20
func (h *MarketHandler) ListMarkets(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	marketType := c.DefaultQuery("type", "sports")

	filter := repository.MarketFilter{
		Type:     marketType, // 一期固定
		Status:   status,
		Platform: "", // 一期不按平台过滤
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
