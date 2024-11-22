package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/morzik45/go-queue/internal/configs"
	"github.com/morzik45/go-queue/internal/db"
	"github.com/morzik45/go-queue/internal/server"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
)

func run(ctx context.Context, _ io.Writer, _ []string) error {
	config := configs.GetConfig(ctx)

	store, err := db.NewMongoDB(ctx, config.Sub("mongodb"))
	if err != nil {
		return err
	}
	srv := server.NewServer(ctx, config, store)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.GetString("web.host"), strconv.Itoa(config.GetInt("web.port"))),
		Handler: srv,
	}

	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_, _ = fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err = httpServer.Shutdown(ctx); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
		err = store.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error closing db: %s\n", err)
		}
	}()

	wg.Wait()
	return nil
}

func main() {
	var ctx context.Context
	ctx = context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := run(ctx, os.Stdout, os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
