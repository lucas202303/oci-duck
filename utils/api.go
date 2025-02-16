package utils

// 导入
import (
	"github.com/gin-gonic/gin"
)

// 版本信息
const (
	CilentVersion    = "1.0.0-Beta-3"
	CilentUpdateTime = "2023.02.20"
)

// Api 检查连接
func ApiInfo(content *gin.Context) {
	data := map[string]interface{}{
		"version":     CilentVersion,
		"msg":         "success",
		"code":        0,
		"update_time": CilentUpdateTime,
	}
	content.JSON(200, data)
}