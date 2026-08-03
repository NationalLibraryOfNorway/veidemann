package archivingcache

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/miekg/dns"
	"github.com/olric-data/olric"
)

var ErrKeyNotFound = errors.New("key not found")

var cacheEntryMagic = [4]byte{'D', 'N', 'S', 3}

type Cachier interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Close(ctx context.Context) error
}

type OlricCache struct {
	client olric.Client
	dmap   olric.DMap
}

func NewOlricCache(addresses []string, dmapName string) (*OlricCache, error) {
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

func (c *OlricCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("cache TTL must be greater than zero: %s", ttl)
	}
	return c.dmap.Put(ctx, key, value, olric.PX(ttl))
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
	StoredAt  time.Time
	ExpiresAt time.Time
	ProxyAddr string
	Msg       *dns.Msg
}

func (ce *CacheEntry) pack() ([]byte, error) {
	if ce.Msg == nil {
		return nil, errors.New("cannot pack cache entry without a DNS message")
	}

	msg, err := ce.Msg.Pack()
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if _, err := buf.Write(cacheEntryMagic[:]); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, ce.StoredAt.UTC().UnixNano()); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, ce.ExpiresAt.UTC().UnixNano()); err != nil {
		return nil, err
	}
	if err := writeBytes(buf, []byte(ce.ProxyAddr)); err != nil {
		return nil, err
	}
	if err := writeBytes(buf, msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ce *CacheEntry) unpack(data []byte) error {
	r := bytes.NewReader(data)

	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return fmt.Errorf("failed to read cache entry version: %w", err)
	}
	if magic != cacheEntryMagic {
		return fmt.Errorf("unsupported cache entry version")
	}

	var storedAt int64
	if err := binary.Read(r, binary.BigEndian, &storedAt); err != nil {
		return fmt.Errorf("failed to read cache entry stored time: %w", err)
	}
	var expiresAt int64
	if err := binary.Read(r, binary.BigEndian, &expiresAt); err != nil {
		return fmt.Errorf("failed to read cache entry expiry: %w", err)
	}

	proxyAddr, err := readBytes(r)
	if err != nil {
		return fmt.Errorf("failed to read cache entry proxy address: %w", err)
	}

	packedMsg, err := readBytes(r)
	if err != nil {
		return fmt.Errorf("failed to read cached DNS message: %w", err)
	}
	if r.Len() != 0 {
		return errors.New("unexpected trailing data in cache entry")
	}

	msg := new(dns.Msg)
	if err := msg.Unpack(packedMsg); err != nil {
		return err
	}
	msg.Authoritative = false

	ce.StoredAt = time.Unix(0, storedAt).UTC()
	ce.ExpiresAt = time.Unix(0, expiresAt).UTC()
	ce.ProxyAddr = string(proxyAddr)
	ce.Msg = msg
	return nil
}

func writeBytes(w io.Writer, data []byte) error {
	if uint64(len(data)) > uint64(^uint32(0)) {
		return errors.New("cache entry field is too large")
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readBytes(r *bytes.Reader) ([]byte, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	if uint64(size) > uint64(r.Len()) {
		return nil, io.ErrUnexpectedEOF
	}
	data := make([]byte, size)
	_, err := io.ReadFull(r, data)
	return data, err
}

func (ce *CacheEntry) String() string {
	return fmt.Sprintf("proxy: %s, stored: %s, expires: %s", ce.ProxyAddr, ce.StoredAt, ce.ExpiresAt)
}
