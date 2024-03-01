package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cockroachdb/pebble"
	badger "github.com/dgraph-io/badger/v4"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type (
	Storage interface {
		Get(id []byte) []byte
		Has(id []byte) bool
		Set(key []byte, value []byte) error
		Scan(prefix []byte) [][]byte
	}
)

type (
	EtcdStorageOptsFunc func(*EtcdStorageOptions)
	EtcdStorageOptions  struct {
		advertiseUrls []string
		timeout       time.Duration
	}
	EtcdStorage struct {
		client  *clientv3.Client
		timeout time.Duration
	}
)

func defaultEtcdOptions() *EtcdStorageOptions {
	url := os.Getenv("ETCD_URL")
	if url == "" {
		url = "http://localhost:2379"
	}
	return &EtcdStorageOptions{
		advertiseUrls: []string{url},
		timeout:       5 * time.Second,
	}
}

type PebbleStorageOptsFunc func(*PebbleStorageOptions)

type PebbleStorageOptions struct {
	path string
}

type BadgerStorageOptsFunc func(*BadgerStorageOptions)

type BadgerStorageOptions struct {
	path string
}

func DefaultBadgerOptions() *BadgerStorageOptions {
	return &BadgerStorageOptions{
		path: "sheshat/badger_storage",
	}
}

func defaultPebbleOptions() *PebbleStorageOptions {
	return &PebbleStorageOptions{
		path: "sheshat/pebble_storage",
	}
}

type PebbleStorage struct {
	handle *pebble.DB
}

func (p *PebbleStorage) Get(id []byte) []byte {
	value, closer, err := p.handle.Get(id)
	if err != nil {
		log.Debug(err)
		return []byte("")
	}
	_ = closer.Close()

	return value
}

func (p *PebbleStorage) Has(id []byte) bool {
	_, closer, err := p.handle.Get(id)
	if err != nil {
		log.Debug(err)
		return false
	}

	_ = closer.Close()

	return true
}

func (p *PebbleStorage) Set(id []byte, value []byte) error {
	if err := p.handle.Set(id, value, pebble.Sync); err != nil {
		log.Error(err)
		return fmt.Errorf("failed to set value at key : %s (%s)", string(id), err)
	}

	return nil
}

func (p *PebbleStorage) Scan(prefix []byte) [][]byte {
	keyUpperBound := func(b []byte) []byte {
		end := make([]byte, len(b))
		copy(end, b)
		for i := len(end) - 1; i >= 0; i-- {
			end[i] = end[i] + 1
			if end[i] != 0 {
				return end[:i+1]
			}
		}
		return nil // no upper-bound
	}

	var results [][]byte

	iter := p.handle.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: keyUpperBound(prefix),
	})
	for iter.First(); iter.Valid(); iter.Next() {
		results = append(results, iter.Value())
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}

	return results
}

// Badger Storage implementation
type BadgerStorage struct {
	handle *badger.DB
}

func (b *BadgerStorage) Get(id []byte) []byte {
	var val []byte
	err := b.handle.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id)
		if err != nil {
			return err
		}

		val, err = item.ValueCopy(val)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to get value from badger storage", "error", err)
		return nil
	}

	return val
}

func (b *BadgerStorage) Has(id []byte) bool {
	err := b.handle.View(func(txn *badger.Txn) error {
		_, err := txn.Get(id)
		if err != nil {
			return err
		}

		return nil
	})

	return !errors.Is(err, badger.ErrKeyNotFound)
}

func (b *BadgerStorage) Set(key []byte, value []byte) error {
	return b.handle.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})
}

func (b *BadgerStorage) Scan(prefix []byte) [][]byte {
	var value [][]byte
	err := b.handle.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var itemVal []byte
			item := it.Item()

			itemVal, _ = item.ValueCopy(itemVal)
			value = append(value, itemVal)
		}
		return nil
	})
	if err != nil {
		log.Error("failed to scan from badger", "error", err)
	}
	return value
}

func (e *EtcdStorage) Get(id []byte) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	resp, err := e.client.Get(ctx, string(id))
	cancel()
	if err != nil {
		log.Error("failed to get value from etcd storage", "error", err)
		return nil
	}

	if resp.Count == 0 {
		return nil
	}

	return resp.Kvs[0].Value
}

func (e *EtcdStorage) Has(id []byte) bool {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	resp, err := e.client.Get(ctx, string(id))
	cancel()
	if err != nil {
		log.Error("failed to get value from etcd storage", "error", err)
		return false
	}

	return resp.Count > 0
}

func (e *EtcdStorage) Set(key []byte, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	_, err := e.client.Put(ctx, string(key), string(value))
	cancel()
	if err != nil {
		log.Error("failed to get value from etcd storage", "error", err)
		return err
	}

	return nil
}

func (e *EtcdStorage) Scan(prefix []byte) [][]byte {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	resp, err := e.client.Get(ctx, string(prefix), clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Error("failed to get value from etcd storage", "error", err)
		return nil
	}

	var value [][]byte
	for _, kv := range resp.Kvs {
		value = append(value, kv.Value)
	}

	return value
}

func NewEtcdStorage(opts ...EtcdStorageOptsFunc) *EtcdStorage {
	o := defaultEtcdOptions()
	for _, optFn := range opts {
		optFn(o)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   o.advertiseUrls,
		DialTimeout: o.timeout,
	})
	if err != nil {
		log.Fatal("failed to get connection to etcd client", "error", err)
		return nil
	}

	return &EtcdStorage{
		client:  cli,
		timeout: o.timeout,
	}
}

func NewBadgerStorage(opts ...BadgerStorageOptsFunc) *BadgerStorage {
	o := DefaultBadgerOptions()
	for _, optFn := range opts {
		optFn(o)
	}

	db, err := badger.Open(badger.DefaultOptions(o.path))
	if err != nil {
		log.Fatal(err)
	}

	return &BadgerStorage{
		handle: db,
	}
}

func NewPebbleStorage(opts ...PebbleStorageOptsFunc) *PebbleStorage {
	o := defaultPebbleOptions()
	for _, optFn := range opts {
		optFn(o)
	}

	handle, err := pebble.Open(o.path, &pebble.Options{})
	if err != nil {
		log.Error(err)
		panic(err)
	}

	return &PebbleStorage{
		handle: handle,
	}
}
