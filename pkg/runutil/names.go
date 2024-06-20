package runutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
)

// NamedWorker assigns a new logutil subsystem on startup. See logutil.Start.
func NamedWorker(worker Worker, name string, a ...any) Worker {
	return WorkerFunc(func(ctx context.Context) error {
		ctx = logutil.Start(ctx, fmt.Sprintf(name, a...))
		return worker.Run(ctx)
	})
}

// NamedWorkerFromType assigns a new logutil subsystem on startup based on the
// provided type name. See logutil.Start.
func NamedWorkerFromType(worker Worker, t any) Worker {
	name := fmt.Sprintf("%T", t)
	name = strings.Trim(name, "*")
	name = strings.Replace(name, ".", "/", 1)
	return NamedWorker(worker, name)
}
