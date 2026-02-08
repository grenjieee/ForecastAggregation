package adapter

import (
	"ForecastSync/internal/config"
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"fmt"

	"github.com/sirupsen/logrus"
)

type PlatformRegistry struct {
	cfg    *config.Config
	logger *logrus.Logger
	// 存储平台类型→适配器实例的映射
	adapters map[model.PlatformType]interfaces.PlatformAdapter
}

func NewPlatformRegistry(cfg *config.Config, logger *logrus.Logger) *PlatformRegistry {
	r := &PlatformRegistry{
		cfg:      cfg,
		logger:   logger,
		adapters: make(map[model.PlatformType]interfaces.PlatformAdapter),
	}

	// 核心修复：从adapter包的工厂函数注册表创建实例
	r.initAdaptersFromFactories()

	return r
}

// initAdaptersFromFactories 从工厂函数注册表初始化适配器实例
func (r *PlatformRegistry) initAdaptersFromFactories() {
	// 步骤1：打印所有已注册的工厂函数（调试）
	factoryPlatforms := ListFactories()
	r.logger.WithField("factory_platforms", factoryPlatforms).Info("adapter包中已注册的工厂函数")

	// 步骤2：遍历配置中的平台，匹配工厂函数创建实例
	for platformStr, platformCfg := range r.cfg.Platforms {
		platformType := platformStr
		r.logger.WithField("platform", platformType).Info("尝试初始化该平台适配器")

		// 从adapter包获取工厂函数
		factory, ok := GetFactory(platformType)
		if !ok {
			r.logger.WithField("platform", platformType).Error("未找到对应的工厂函数（init未注册？）")
			continue
		}

		// 调用工厂函数创建适配器实例
		adapterIns := factory(&platformCfg, r.logger)
		if adapterIns == nil {
			r.logger.WithField("platform", platformType).Error("工厂函数返回nil适配器实例")
			continue
		}

		// 验证实例的平台类型是否匹配
		if adapterIns.GetType() != platformType {
			r.logger.WithFields(logrus.Fields{
				"config_platform":  platformType,
				"adapter_platform": adapterIns.GetType(),
			}).Error("适配器平台类型与配置不匹配")
			continue
		}

		// 存储到实例注册表
		r.adapters[platformType] = adapterIns
		r.logger.WithField("platform", platformType).Info("适配器实例初始化成功并加入注册表")
	}

	// 步骤3：打印最终实例注册表
	r.logger.WithField("instance_platforms", len(r.adapters)).Info("最终初始化的适配器实例数量")
	for p := range r.adapters {
		r.logger.WithField("platform", p).Info("已初始化的适配器实例")
	}
}

// ListRegisteredPlatforms 获取所有已注册并初始化的平台类型列表
func (r *PlatformRegistry) ListRegisteredPlatforms() []model.PlatformType {
	var platforms []model.PlatformType
	for p := range r.adapters {
		platforms = append(platforms, p)
	}
	return platforms
}

// GetAdapter 获取适配器实例（原有逻辑，现在能拿到值了）
func (r *PlatformRegistry) GetAdapter(platform model.PlatformType) (interfaces.PlatformAdapter, error) {
	// 调试日志
	var registered []string
	for p := range r.adapters {
		registered = append(registered, string(p))
	}
	r.logger.WithFields(logrus.Fields{
		"requested":  platform,
		"registered": registered,
	}).Info("尝试获取适配器实例")

	adapterIns, ok := r.adapters[platform]
	if !ok {
		return nil, fmt.Errorf("平台%s未初始化适配器实例（已初始化：%v）", platform, registered)
	}
	return adapterIns, nil
}

// GetPlatformCount 获取已初始化实例的平台数量
func (r *PlatformRegistry) GetPlatformCount() int {
	return len(r.adapters)
}

// Factory 平台适配器工厂函数签名
// 入参：平台配置、日志实例
// 出参：实现PlatformAdapter接口的适配器实例
type Factory func(cfg *config.PlatformConfig, logger *logrus.Logger) interfaces.PlatformAdapter
