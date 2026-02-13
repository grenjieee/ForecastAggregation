package api

import (
	"net/http"
	"strconv"

	"ForecastSync/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// OrderHandler 订单查询接口
type OrderHandler struct {
	orderService *service.OrderService
	logger       *logrus.Logger
}

// NewOrderHandler 创建 OrderHandler
func NewOrderHandler(db *gorm.DB, logger *logrus.Logger) *OrderHandler {
	svc := service.NewOrderService(db, logger, nil)
	return &OrderHandler{
		orderService: svc,
		logger:       logger,
	}
}

// ListOrders 订单列表 GET /api/orders?wallet=0x...&page=1&page_size=20
func (h *OrderHandler) ListOrders(c *gin.Context) {
	wallet := c.Query("wallet")
	if wallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet is required"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.orderService.ListByUser(c.Request.Context(), wallet, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("ListOrders failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetOrderDetail 订单详情 GET /api/orders/:order_uuid
func (h *OrderHandler) GetOrderDetail(c *gin.Context) {
	orderUUID := c.Param("order_uuid")
	if orderUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uuid is required"})
		return
	}

	result, err := h.orderService.GetOrderDetail(c.Request.Context(), orderUUID)
	if err != nil {
		h.logger.WithError(err).Error("GetOrderDetail failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
