package cmd

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/redisutil"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/webutil"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	RedisClient *redis.Client
	RedisPrefix redisutil.Prefix
}

func (s *Server) Run(ctxRoot context.Context) error {
	// Creating a new context, so we can have two stages for the graceful
	// shutdown. First is to make pod unready (within the admin api) and the
	// seconds is all the rest.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Using a errors group is a good practice to manage multiple parallel
	// running routines and should used once on program startup. We have to use
	// ctxRoot, because this is what should canceled first, if any error
	// occours.
	group, ctxRoot := errgroup.WithContext(ctxRoot)

	webutil.AdminAPIListenAndServe(ctxRoot, group, cancel)

	go func() {
		<-ctx.Done()
		fmt.Println("tschüss")
	}()

	return errors.WithStack(group.Wait())
}
