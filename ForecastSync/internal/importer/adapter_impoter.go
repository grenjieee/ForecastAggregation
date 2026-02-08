package importer

/**
这个文件只是为了解决循环依赖，保证各个平台的init方法一定会被触发
*/
import _ "ForecastSync/internal/adapter/polymarket"
import _ "ForecastSync/internal/adapter/kalshi"
