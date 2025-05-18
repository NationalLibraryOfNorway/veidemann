package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/olric-data/olric"
)

var ErrKeyNotFound = errors.New("key not found")

type Cachier interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, value []byte) error
	Close(ctx context.Context) error
}

type OlricCache struct {
	client *olric.ClusterClient
	dmap   olric.DMap
}

func NewOlricCache(addresses []string, dmapName string) (*OlricCache, error) {
	client, err := olric.NewClusterClient(addresses)
	if err != nil {
		return nil, fmt.Errorf("failed to create Olric client: %w", err)
	}

	dmap, err := client.NewDMap(dmapName)
	if err != nil {
		return nil, fmt.Errorf("failed to create DMap: %w", err)
	}

	return &OlricCache{
		client: client,
		dmap:   dmap,
	}, nil
}

func (c *OlricCache) Get(ctx context.Context, key string) ([]byte, error) {
	gr, err := c.dmap.Get(ctx, key)
	if errors.Is(err, olric.ErrKeyNotFound) {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return gr.Byte()
}

func (c *OlricCache) Put(ctx context.Context, key string, value []byte) error {
	return c.dmap.Put(ctx, key, value)
}

func (c *OlricCache) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}
