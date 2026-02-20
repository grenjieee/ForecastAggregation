package api

import (
	"net/http"
	"strconv"

	"ForecastSync/internal/circle"
	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NewOrderHandler 创建 OrderHandler。adapters 为 nil 时仅支持查询，PlaceOrder 会报错
// cfg 用于构建 Circle 兑换服务（Kalshi 下单前链资产转 USD）
func NewOrderHandler(db *gorm.DB, logger *logrus.Logger, adapters map[uint64]interfaces.TradingAdapter, cfg *config.Config) *OrderHandler {
	var fiat service.FiatConversionService
	if cfg != nil && cfg.Circle.APIKey != "" && cfg.Circle.BaseURL != "" {
		circleClient := circle.NewClient(circle.Config{
			BaseURL: cfg.Circle.BaseURL,
			APIKey:  cfg.Circle.APIKey,
			Timeout: cfg.Circle.Timeout,
			Proxy:   cfg.Circle.Proxy,
		}, logger)
		fiat = service.NewCircleFiatConversion(circleClient)
		logger.Info("OrderHandler 使用 Circle 兑换服务")
	} else {
		fiat = service.NewNoopFiatConversion()
		logger.Info("OrderHandler 使用占位兑换（未配置 Circle API Key）")
	}
	svc := service.NewOrderServiceWithDeps(db, logger, adapters, fiat)
	return &OrderHandler{
		orderService: svc,
		logger:       logger,
	}
}

// OrderHandler 订单查询与下单接口
type OrderHandler struct {
	orderService *service.OrderService
	logger       *logrus.Logger
}

// ListOrders 订单列表 GET /api/orders?wallet=0x...&page=1&page_size=20&status=settled
// status 可选：settled=可提现订单
func (h *OrderHandler) ListOrders(c *gin.Context) {
	wallet := c.Query("wallet")
	if wallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet is required"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	result, err := h.orderService.ListByUserWithStatus(c.Request.Context(), wallet, status, page, pageSize)
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

// GetWithdrawInfo 获取提现参数 GET /api/orders/:order_uuid/withdraw-info
func (h *OrderHandler) GetWithdrawInfo(c *gin.Context) {
	orderUUID := c.Param("order_uuid")
	if orderUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uuid is required"})
		return
	}
	result, err := h.orderService.GetWithdrawInfo(c.Request.Context(), orderUUID)
	if err != nil {
		h.logger.WithError(err).Error("GetWithdrawInfo failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// RequestWithdraw 发起提现 POST /api/orders/:order_uuid/withdraw
func (h *OrderHandler) RequestWithdraw(c *gin.Context) {
	orderUUID := c.Param("order_uuid")
	if orderUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uuid is required"})
		return
	}
	if err := h.orderService.RequestWithdraw(c.Request.Context(), orderUUID); err != nil {
		h.logger.WithError(err).Error("RequestWithdraw failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "提现请求已记录"})
}

// PlaceOrder 下单接口 POST /api/orders/place
func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	var req service.PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	result, err := h.orderService.PlaceOrderFromFrontend(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("PlaceOrder failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
