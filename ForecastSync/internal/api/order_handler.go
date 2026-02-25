package api

import (
	"net/http"
	"strconv"

	"ForecastSync/internal/adapter/kalshi"
	"ForecastSync/internal/adapter/polymarket"
	"ForecastSync/internal/circle"
	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/repository"
	"ForecastSync/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NewOrderHandler 创建 OrderHandler。adapters 为 nil 时仅支持查询，PlaceOrder 会报错
// cfg 用于构建 Circle 兑换服务（Kalshi 下单前链资产转 USD）及实时赔率拉取适配器
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
	eventRepo := repository.NewEventRepositoryInstance(db)
	liveOddsFetchers := make(map[uint64]interfaces.LiveOddsFetcher)
	if cfg != nil {
		if p, ok := cfg.Platforms["polymarket"]; ok {
			if lf, ok := polymarket.NewPolymarketAdapter(&p, logger).(interfaces.LiveOddsFetcher); ok {
				liveOddsFetchers[1] = lf
			}
		}
		if k, ok := cfg.Platforms["kalshi"]; ok {
			if lf, ok := kalshi.NewKalshiAdapter(&k, logger).(interfaces.LiveOddsFetcher); ok {
				liveOddsFetchers[2] = lf
			}
		}
	}
	var chainCfg *config.ChainConfig
	if cfg != nil {
		chainCfg = &cfg.Chain
	}
	svc := service.NewOrderServiceWithDeps(db, logger, adapters, fiat, eventRepo, liveOddsFetchers, chainCfg)
	return &OrderHandler{
		orderService: svc,
		cfg:          cfg,
		logger:       logger,
	}
}

// OrderHandler 订单查询与下单接口
type OrderHandler struct {
	orderService *service.OrderService
	cfg          *config.Config
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

// PrepareOrder 获取待签名信息（实时查三方赔率，返回最高赔率与待签名消息）POST /api/orders/prepare
func (h *OrderHandler) PrepareOrder(c *gin.Context) {
	var req service.PrepareOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	result, err := h.orderService.PrepareOrderFromFrontend(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("PrepareOrder failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// PlaceOrder 下单接口 POST /api/orders/place（可选带 message_to_sign + signature，校验通过后才真实下单）
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

// UnfreezeRequest 解冻请求 body
type UnfreezeRequest struct {
	ContractOrderID string `json:"contract_order_id"` // 必填
	Wallet          string `json:"wallet"`            // 可选，校验与入账钱包一致
}

// RequestUnfreeze 申请解冻 POST /api/orders/unfreeze
func (h *OrderHandler) RequestUnfreeze(c *gin.Context) {
	var req UnfreezeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	txHash, err := h.orderService.RequestUnfreeze(c.Request.Context(), req.ContractOrderID, req.Wallet)
	if err != nil {
		h.logger.WithError(err).Error("RequestUnfreeze failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash})
}

// GetContractOrderStatus 合约订单状态 GET /api/orders/contract-order-status?contract_order_id=xxx
func (h *OrderHandler) GetContractOrderStatus(c *gin.Context) {
	contractOrderID := c.Query("contract_order_id")
	if contractOrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contract_order_id is required"})
		return
	}
	status, err := h.orderService.ContractOrderStatus(c.Request.Context(), contractOrderID)
	if err != nil {
		h.logger.WithError(err).Error("ContractOrderStatus failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}
