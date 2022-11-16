# gin-grpc-network
一个可以同时处理http和grpc请求的网络库

# 例子
[完整的例子](https://github.com/DAN-AND-DNA/easyman)

 - 写一个handler就可以处理grpc和http的请求:  
```golang
func setupHandlers() {
	// grpc 以pkg service method来区别请求
	network.HandleProto("webbff", "webbff", "login", &webbff.WebBFF_ServiceDesc, gingrpc.Handler{
		Proto: &webbff.LoginReq{},
		HandleProto: func(ctx context.Context, req interface{}) (interface{}, error) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			l := ctxzap.Extract(ctx)
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, status.Error(codes.Internal, "")
			}

			ct := md.Get("content-type")
			if len(ct) == 0 {
				return nil, status.Error(codes.NotFound, "content-type lost")
			}

			l.Info("get Content-Type", zap.String("Content-Type", ct[0]))

			reqProto := req.(*webbff.LoginReq)
			username := reqProto.GetName()
			password := reqProto.GetPassword()

			l.Info("user try to login", zap.String("name", username), zap.String("password", password))

			return &webbff.LoginResp{Token: "xxxxxxxx"}, nil
		},
	})
}
```

## http性能
```c++
goos: windows
goarch: amd64
pkg: github.com/dan-and-dna/gin-grpc
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
BenchmarkGinGrpc
BenchmarkGinGrpc-12      3528080              1675 ns/op            1496 B/op
              13 allocs/op
PASS
```
 
## grpc性能
中间件，故无额外开销，仅为grpc-go的开销
```c++
goos: windows
goarch: amd64
pkg: easyman
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
BenchmarkGrpc
BenchmarkGrpc-12          112734             10224 ns/op            1702 B/op           36 allocs/op
PASS
```
