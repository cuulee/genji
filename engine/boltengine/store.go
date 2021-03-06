package boltengine

import (
	"bytes"
	"context"

	"github.com/genjidb/genji/engine"
	bolt "go.etcd.io/bbolt"
)

// A Store is an implementation of the engine.Store interface using a bucket.
type Store struct {
	bucket *bolt.Bucket
	tx     *bolt.Tx
	name   []byte
	ctx    context.Context
}

// Put stores a key value pair. If it already exists, it overrides it.
func (s *Store) Put(k, v []byte) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	if !s.bucket.Writable() {
		return engine.ErrTransactionReadOnly
	}

	return s.bucket.Put(k, v)
}

// Get returns a value associated with the given key. If not found, returns engine.ErrKeyNotFound.
func (s *Store) Get(k []byte) ([]byte, error) {
	select {
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	default:
	}

	v := s.bucket.Get(k)
	if v == nil {
		return nil, engine.ErrKeyNotFound
	}

	return v, nil
}

// Delete a record by key. If not found, returns table.ErrDocumentNotFound.
func (s *Store) Delete(k []byte) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	if !s.bucket.Writable() {
		return engine.ErrTransactionReadOnly
	}

	v := s.bucket.Get(k)
	if v == nil {
		return engine.ErrKeyNotFound
	}

	return s.bucket.Delete(k)
}

// Truncate deletes all the records of the store.
func (s *Store) Truncate() error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	if !s.bucket.Writable() {
		return engine.ErrTransactionReadOnly
	}

	err := s.tx.DeleteBucket(s.name)
	if err != nil {
		return err
	}

	_, err = s.tx.CreateBucket(s.name)
	return err
}

// NextSequence returns a monotonically increasing integer.
func (s *Store) NextSequence() (uint64, error) {
	select {
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	default:
	}

	if !s.bucket.Writable() {
		return 0, engine.ErrTransactionReadOnly
	}

	return s.bucket.NextSequence()
}

// Iterator uses the Bolt bucket cursor.
func (s *Store) Iterator(opts engine.IteratorOptions) engine.Iterator {
	return &iterator{
		c:       s.bucket.Cursor(),
		reverse: opts.Reverse,
		ctx:     s.ctx,
	}
}

type iterator struct {
	c       *bolt.Cursor
	reverse bool
	item    boltItem
	err     error
	ctx     context.Context
}

func (it *iterator) Seek(pivot []byte) {
	select {
	case <-it.ctx.Done():
		it.err = it.ctx.Err()
		return
	default:
	}

	if !it.reverse {
		it.item.k, it.item.v = it.c.Seek(pivot)
		return
	}

	if len(pivot) == 0 {
		it.item.k, it.item.v = it.c.Last()
		return
	}

	it.item.k, it.item.v = it.c.Seek(pivot)
	if it.item.k != nil {
		for bytes.Compare(it.item.k, pivot) > 0 {
			it.item.k, it.item.v = it.c.Prev()
		}
	}
}

func (it *iterator) Valid() bool {
	return it.item.k != nil && it.err == nil
}

func (it *iterator) Next() {
	if it.reverse {
		it.item.k, it.item.v = it.c.Prev()
	} else {
		it.item.k, it.item.v = it.c.Next()
	}
}

func (it *iterator) Err() error {
	return it.err
}

func (it *iterator) Item() engine.Item {
	if it.item.k == nil {
		return nil
	}

	return &it.item
}

func (it *iterator) Close() error { return nil }

type boltItem struct {
	k, v []byte
}

func (i *boltItem) Key() []byte {
	return i.k
}

func (i *boltItem) ValueCopy(buf []byte) ([]byte, error) {
	return append(buf[:0], i.v...), nil
}
