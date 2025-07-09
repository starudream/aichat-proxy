package browser

import (
	"context"
	"sync"
)

func Start(ctx context.Context, wg *sync.WaitGroup) {
	startProxy(ctx, wg)
	startBrowser(ctx, wg)
}
