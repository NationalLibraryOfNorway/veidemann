package archivingcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/miekg/dns"
	"github.com/olric-data/olric"
)

var ErrKeyNotFound = errors.New("key not found")

type Cachier interface {
	Get(ctx context.Context, key string) (*CacheEntry, error)
	Set(ctx context.Context, key string, entry *CacheEntry) error
	Len(ctx context.Context) (int, error)
	Close(ctx context.Context) error
}

type OlricCache struct {
	client olric.Client
	dmap   olric.DMap
	ttl    time.Duration
}

func NewOlricCache(addresses []string, dmapName string, ttl time.Duration) (*OlricCache, error) {
	client, err := olric.NewClusterClient(addresses)
	if err != nil {
		return nil, fmt.Errorf("failed to create Olric client: %w", err)
	}

	dmap, err := client.NewDMap(dmapName)
	if err != nil {
		_ = client.Close(context.Background())
		return nil, fmt.Errorf("failed to create DMap: %w", err)
	}

	return &OlricCache{
		client: client,
		dmap:   dmap,
		ttl:    ttl,
	}, nil
}

func (c *OlricCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	gr, err := c.dmap.Get(ctx, key)
	if errors.Is(err, olric.ErrKeyNotFound) {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	data, err := gr.Byte()
	if err != nil {
		return nil, err
	}

	entry := new(CacheEntry)
	if err := entry.unpack(data); err != nil {
		return nil, err
	}

	return entry, nil
}

func (c *OlricCache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	packed, err := entry.pack()
	if err != nil {
		return err
	}

	if c.ttl > 0 {
		return c.dmap.Put(ctx, key, packed, olric.PX(c.ttl))
	}
	return c.dmap.Put(ctx, key, packed)
}

func (c *OlricCache) Len(ctx context.Context) (int, error) {
	iter, err := c.dmap.Scan(ctx)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		count++
	}
	return count, nil
}

func (c *OlricCache) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.dmap != nil {
		_ = c.dmap.Close(ctx)
	}
	if c.client != nil {
		return c.client.Close(ctx)
	}
	return nil
}

type CacheEntry struct {
	ProxyAddr     string
	CollectionIds []string
	Msg           *dns.Msg
}

func (ce *CacheEntry) pack() ([]byte, error) {
	var packed []byte

	// proxy addr
	if len(ce.ProxyAddr) > 0 {
		packed = append(packed, ce.ProxyAddr...)
		packed = append(packed, '|')
	}

	// collection ids
	for _, v := range ce.CollectionIds {
		packed = append(packed, v...)
		packed = append(packed, ':')
	}
	packed = append(packed, ':')

	// dns message
	entry, err := ce.Msg.Pack()
	if err != nil {
		return nil, err
	}
	packed = append(packed, entry...)
	return packed, nil
}

func (ce *CacheEntry) unpack(entry []byte) error {
	// proxy address
	idx := bytes.IndexByte(entry, '|')
	if idx != -1 {
		ce.ProxyAddr = string(entry[:idx])
		entry = entry[idx+1:]
	}

	// collection ids
	for entry[0] != ':' {
		idx := bytes.IndexByte(entry, ':')
		if idx == -1 {
			return fmt.Errorf("error unpacking collections from cache entry")
		}
		ce.CollectionIds = append(ce.CollectionIds, string(entry[:idx]))
		entry = entry[idx+1:]
	}
	entry = entry[1:]

	// dns message
	m := new(dns.Msg)
	err := m.Unpack(entry)
	if err != nil {
		return err
	}

	m.Authoritative = false
	ce.Msg = m
	return nil
}

func (ce *CacheEntry) AddCollectionId(collectionId string) []string {
	ce.CollectionIds = append(ce.CollectionIds, collectionId)
	return ce.CollectionIds
}

func (ce *CacheEntry) HasCollectionId(collectionId string) bool {
	if collectionId == "" {
		return true
	}
	for _, cid := range ce.CollectionIds {
		if cid == collectionId {
			return true
		}
	}
	return false
}

func (ce *CacheEntry) String() string {
	return fmt.Sprintf("proxy: %s, collections: %v", ce.ProxyAddr, ce.CollectionIds)
}
