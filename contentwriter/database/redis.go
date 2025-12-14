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
	// Collection string
	// TTL        time.Duration // configurable; can change at runtime if you update c.TTL
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

func (c *CrawledContentHashCache) WriteCrawledContent(ctx context.Context, collection string, ttl time.Duration, crawledContent *contentwriterV1.CrawledContent) error {
	if crawledContent == nil {
		return errors.New("missing required field: crawledContent")
	}

	b, err := proto.Marshal(crawledContent)
	if err != nil {
		return err
	}
	return c.writeCrawledContent(ctx, collection, crawledContent.Digest, b, ttl)
}

// WriteCrawledContent upserts the field and refreshes the collection TTL (sliding TTL).
// This also naturally applies TTL config changes on the next write.
func (c *CrawledContentHashCache) writeCrawledContent(ctx context.Context, key, field string, value []byte, ttl time.Duration) error {
	if field == "" {
		return errors.New("missing required field: field")
	}
	if len(value) == 0 {
		return errors.New("missing required field: value")
	}

	if ttl <= 0 {
		_, err := c.Client.HSet(ctx, key, field, value).Result()
		return err
	}

	pipe := c.Client.Pipeline()
	pipe.HSet(ctx, key, field, value)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}
