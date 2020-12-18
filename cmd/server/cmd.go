package main

import (
	"context"
	"fmt"
	"log"

	"github.com/whywaita/myshoes/internal/config"
	"github.com/whywaita/myshoes/pkg/datastore"
	"github.com/whywaita/myshoes/pkg/datastore/mysql"
	"github.com/whywaita/myshoes/pkg/runner"
	"github.com/whywaita/myshoes/pkg/starter"
	"github.com/whywaita/myshoes/pkg/starter/safety/unlimited"
	"github.com/whywaita/myshoes/pkg/web"

	"golang.org/x/sync/errgroup"
)

func init() {
	config.Load()
}

func main() {
	myshoes, err := newShoes()
	if err != nil {
		log.Fatalln(err)
	}

	if err := myshoes.Run(); err != nil {
		log.Fatalln(err)
	}
}

type myShoes struct {
	ds    datastore.Datastore
	start *starter.Starter
	run   *runner.Manager
}

// newShoes create myshoes.
func newShoes() (*myShoes, error) {
	ds, err := mysql.New(config.Config.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to mysql.New: %w", err)
	}

	unlimit := unlimited.Unlimited{}
	s := starter.New(ds, unlimit)

	manager := runner.New(ds)

	return &myShoes{
		ds:    ds,
		start: s,
		run:   manager,
	}, nil
}

// Run start services.
func (m *myShoes) Run() error {
	eg := errgroup.Group{}
	ctx := context.Background()

	eg.Go(func() error {
		if err := web.Serve(m.ds); err != nil {
			return fmt.Errorf("failed to serve: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := m.start.Loop(ctx); err != nil {
			return fmt.Errorf("failed to starter loop: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := m.run.Loop(ctx); err != nil {
			return fmt.Errorf("failed to runner loop: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to wait errgroup: %w", err)
	}

	return nil
}
