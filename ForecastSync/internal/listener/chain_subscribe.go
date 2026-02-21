package listener

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"ForecastSync/internal/config"
	"ForecastSync/internal/service"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

const usdcDecimals = 6

var (
	// FundsLocked(bytes32 indexed betId, address from, uint256 amount)
	sigFundsLocked = crypto.Keccak256Hash([]byte("FundsLocked(bytes32,address,uint256)"))
	// Settled(bytes32 indexed betId, uint256 payout, uint256 fee)
	sigSettled = crypto.Keccak256Hash([]byte("Settled(bytes32,uint256,uint256)"))
)

// ChainSubscriber 使用 go-ethereum 订阅链上事件并回调 ContractListener
type ChainSubscriber struct {
	cfg      *config.ChainConfig
	client   *ethclient.Client
	listener *ContractListener
	logger   *logrus.Logger
}

// NewChainSubscriber 创建链上订阅器（需传入已连接的 ethclient，便于测试）
func NewChainSubscriber(cfg *config.ChainConfig, client *ethclient.Client, listener *ContractListener, logger *logrus.Logger) *ChainSubscriber {
	return &ChainSubscriber{cfg: cfg, client: client, listener: listener, logger: logger}
}

// Run 在后台订阅 Escrow.FundsLocked 与 Settlement.Settled，解析后调用 listener
func (s *ChainSubscriber) Run(ctx context.Context) error {
	if s.cfg.EscrowAddress == "" || s.cfg.SettlementAddress == "" {
		s.logger.Info("ChainSubscriber: escrow_address 或 settlement_address 未配置，跳过订阅")
		<-ctx.Done()
		return nil
	}
	escrowAddr := common.HexToAddress(s.cfg.EscrowAddress)
	settlementAddr := common.HexToAddress(s.cfg.SettlementAddress)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{escrowAddr, settlementAddr},
		Topics:    [][]common.Hash{{sigFundsLocked, sigSettled}},
	}
	ch := make(chan types.Log)
	sub, err := s.client.SubscribeFilterLogs(ctx, query, ch)
	if err != nil {
		return fmt.Errorf("SubscribeFilterLogs: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			s.logger.WithError(err).Error("ChainSubscriber subscription error")
			return err
		case vLog := <-ch:
			if err := s.handleLog(ctx, vLog, escrowAddr, settlementAddr); err != nil {
				s.logger.WithError(err).WithField("tx_hash", vLog.TxHash.Hex()).Warn("handleLog failed")
			}
		}
	}
}

func (s *ChainSubscriber) handleLog(ctx context.Context, vLog types.Log, escrowAddr, settlementAddr common.Address) error {
	switch {
	case vLog.Address == escrowAddr && len(vLog.Topics) > 0 && vLog.Topics[0] == sigFundsLocked:
		return s.handleFundsLocked(ctx, vLog)
	case vLog.Address == settlementAddr && len(vLog.Topics) > 0 && vLog.Topics[0] == sigSettled:
		return s.handleSettled(ctx, vLog)
	default:
		return nil
	}
}

func (s *ChainSubscriber) handleFundsLocked(ctx context.Context, vLog types.Log) error {
	// topic1 = betId (indexed bytes32)
	if len(vLog.Topics) < 2 {
		return fmt.Errorf("FundsLocked missing topic betId")
	}
	betId := vLog.Topics[1]
	contractOrderID := "0x" + hex.EncodeToString(betId.Bytes())
	// Data: from (address) + amount (uint256) = 32+32 bytes
	if len(vLog.Data) < 64 {
		return fmt.Errorf("FundsLocked data too short")
	}
	fromAddr := common.BytesToAddress(vLog.Data[12:32])
	amountBig := new(big.Int).SetBytes(vLog.Data[32:64])
	amount := amountToFloat(amountBig, usdcDecimals)
	ev := &service.DepositSuccessEvent{
		ContractOrderID: strings.TrimPrefix(contractOrderID, "0x"),
		UserWallet:      fromAddr.Hex(),
		Amount:          amount,
		Currency:        "USDC",
		TxHash:          vLog.TxHash.Hex(),
		BlockNumber:     int64(vLog.BlockNumber),
		RawData:         nil,
	}
	return s.listener.OnDepositSuccess(ctx, ev)
}

func (s *ChainSubscriber) handleSettled(ctx context.Context, vLog types.Log) error {
	if len(vLog.Topics) < 2 {
		return fmt.Errorf("Settled missing topic betId")
	}
	betId := vLog.Topics[1]
	orderUUID := hex.EncodeToString(betId.Bytes())
	if len(vLog.Data) < 64 {
		return fmt.Errorf("Settled data too short")
	}
	payoutBig := new(big.Int).SetBytes(vLog.Data[0:32])
	feeBig := new(big.Int).SetBytes(vLog.Data[32:64])
	payout := amountToFloat(payoutBig, usdcDecimals)
	fee := amountToFloat(feeBig, usdcDecimals)
	return s.listener.OnSettlementCompleted(ctx, orderUUID, vLog.TxHash.Hex(), payout, fee, 0)
}

func amountToFloat(b *big.Int, decimals int) float64 {
	if b == nil {
		return 0
	}
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	var f, divF big.Float
	f.SetInt(b)
	divF.SetInt(div)
	f.Quo(&f, &divF)
	q, _ := f.Float64()
	return q
}
