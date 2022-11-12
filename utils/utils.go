package utils

import "strings"

// 创建grpc的完整服务名
func MakeKey(pkg, service, method string) string {
	return strings.ToLower("/" + pkg + "." + service + "/" + method)
}
