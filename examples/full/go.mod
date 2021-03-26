module github.com/rebuy-de/rebuy-go-sdk/examples/full

go 1.16

replace github.com/rebuy-de/rebuy-go-sdk/v3 => ../..

require (
	github.com/alicebob/miniredis/v2 v2.14.3
	github.com/go-redis/redis/v8 v8.7.1
	github.com/pkg/errors v0.9.1
	github.com/rebuy-de/rebuy-go-sdk/v3 v3.5.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)
