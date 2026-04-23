package database

import (
	"context"
	"errors"
	"time"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type CrawledContentHashCache struct {
	Client *redis.Client
}

// HasCrawledContent returns the stored blob for digest (field) or nil if absent.
func (c *CrawledContentHashCache) HasCrawledContent(ctx context.Context, key, digest string) (*contentwriterV1.CrawledContent, error) {
	b, err := c.hasCrawledContent(ctx, key, digest)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}

	var cc contentwriterV1.CrawledContent
	err = proto.Unmarshal(b, &cc)
	if err != nil {
		return nil, err
	}

	// Restore the digest field since it was removed before storage.
	cc.Digest = digest

	return &cc, nil
}

// HasCrawledContent returns the stored blob for digest (field) or nil if absent.
func (c *CrawledContentHashCache) hasCrawledContent(ctx context.Context, key string, field string) ([]byte, error) {
	if field == "" {
		return nil, errors.New("missing required field: field")
	}
	b, err := c.Client.HGet(ctx, key, field).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return b, err
}

func (c *CrawledContentHashCache) WriteCrawledContent(ctx context.Context, collection string, crawledContent *contentwriterV1.CrawledContent, expiresAt time.Time) error {
	if crawledContent == nil {
		return errors.New("missing required field: crawledContent")
	}

	// Don't need to store the digest inside the stored object because it's already the field name.
	digest := crawledContent.Digest
	crawledContent.Digest = ""

	b, err := proto.Marshal(crawledContent)
	if err != nil {
		return err
	}
	return c.writeCrawledContent(ctx, collection, digest, b, expiresAt)
}

// WriteCrawledContent upserts the field and sets the expiresAt if the key doesn't have a TTL. If the key already has a TTL, it leaves it unchanged.
func (c *CrawledContentHashCache) writeCrawledContent(ctx context.Context, key, field string, value []byte, expiresAt time.Time) error {
	if field == "" {
		return errors.New("missing required field: field")
	}
	if len(value) == 0 {
		return errors.New("missing required field: value")
	}

	err := c.Client.HSet(ctx, key, field, value).Err()
	if err != nil {
		return err
	}

	if expiresAt.IsZero() {
		return nil
	}

	ttl, err := c.Client.TTL(ctx, key).Result()
	if err != nil {
		return err
	}

	if ttl == -1 {
		return c.Client.ExpireAt(ctx, key, expiresAt).Err()
	}

	return nil
}
