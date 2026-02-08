// internal/adapter/registry.go
package adapter

import (
	"ForecastSync/internal/interfaces"
	"ForecastSync/internal/model"
	"fmt"

	"github.com/sirupsen/logrus"
)

// ========== 全局工厂函数注册表（依赖interfaces包） ==========
var factoryRegistry = make(map[model.PlatformType]interfaces.Factory)

// Register 供适配器init函数调用，注册工厂函数（逻辑不变）
func Register(platform model.PlatformType, factory interfaces.Factory) {
	if factory == nil {
		panic(fmt.Sprintf("平台%s的工厂函数不能为nil", platform))
	}
	if _, exists := factoryRegistry[platform]; exists {
		logrus.Warnf("平台%s的适配器已注册，将覆盖原有实现", platform)
	}
	factoryRegistry[platform] = factory
	logrus.Infof("平台%s工厂函数注册成功", platform)
}

// GetFactory 获取指定平台的工厂函数
func GetFactory(platform model.PlatformType) (interfaces.Factory, bool) {
	factory, ok := factoryRegistry[platform]
	return factory, ok
}

// ListFactories 列出所有已注册的工厂函数平台
func ListFactories() []model.PlatformType {
	var platforms []model.PlatformType
	for p := range factoryRegistry {
		platforms = append(platforms, p)
	}
	return platforms
}
