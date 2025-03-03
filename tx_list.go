// Copyright 2019 The nutsdb Author. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nutsdb

import (
	"bytes"
	"sort"
	"strings"
	"time"

	"github.com/nutsdb/nutsdb/ds/list"
	"github.com/pkg/errors"
	"github.com/xujiajun/utils/strconv2"
)

var (
	// ErrSeparatorForListKey returns when list key contains the SeparatorForListKey.
	ErrSeparatorForListKey = errors.Errorf("contain separator (%s) for List key", SeparatorForListKey)
)

// SeparatorForListKey represents separator for listKey
const SeparatorForListKey = "|"

// RPop removes and returns the last element of the list stored in the bucket at given bucket and key.
func (tx *Tx) RPop(bucket string, key []byte) (item []byte, err error) {
	item, err = tx.RPeek(bucket, key)
	if err != nil {
		return
	}

	return item, tx.push(bucket, key, DataRPopFlag, item)
}

// RPeek returns the last element of the list stored in the bucket at given bucket and key.
func (tx *Tx) RPeek(bucket string, key []byte) (item []byte, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return nil, err
	}

	l := tx.db.Index.getList(bucket)
	if l == nil {
		return nil, ErrBucket
	}

	if tx.CheckExpire(bucket, key) {
		return nil, ErrKeyNotFound
	}

	item, _, err = l.RPeek(string(key))

	return
}

// push sets values for list stored in the bucket at given bucket, key, flag and values.
func (tx *Tx) push(bucket string, key []byte, flag uint16, values ...[]byte) error {
	for _, value := range values {
		err := tx.put(bucket, key, value, Persistent, flag, uint64(time.Now().Unix()), DataStructureList)
		if err != nil {
			return err
		}
	}

	return nil
}

// RPush inserts the values at the tail of the list stored in the bucket at given bucket,key and values.
func (tx *Tx) RPush(bucket string, key []byte, values ...[]byte) error {
	if err := tx.checkTxIsClosed(); err != nil {
		return err
	}
	if tx.CheckExpire(bucket, key) {
		return ErrKeyNotFound
	}
	if strings.Contains(string(key), SeparatorForListKey) {
		return ErrSeparatorForListKey
	}

	return tx.push(bucket, key, DataRPushFlag, values...)
}

// LPush inserts the values at the head of the list stored in the bucket at given bucket,key and values.
func (tx *Tx) LPush(bucket string, key []byte, values ...[]byte) error {
	if err := tx.checkTxIsClosed(); err != nil {
		return err
	}
	if tx.CheckExpire(bucket, key) {
		return ErrKeyNotFound
	}

	if strings.Contains(string(key), SeparatorForListKey) {
		return ErrSeparatorForListKey
	}

	return tx.push(bucket, key, DataLPushFlag, values...)
}

// LPop removes and returns the first element of the list stored in the bucket at given bucket and key.
func (tx *Tx) LPop(bucket string, key []byte) (item []byte, err error) {
	item, err = tx.LPeek(bucket, key)
	if err != nil {
		return
	}

	return item, tx.push(bucket, key, DataLPopFlag, item)
}

// LPeek returns the first element of the list stored in the bucket at given bucket and key.
func (tx *Tx) LPeek(bucket string, key []byte) (item []byte, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return nil, err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return nil, ErrBucket
	}
	if tx.CheckExpire(bucket, key) {
		return nil, ErrKeyNotFound
	}
	item, err = l.LPeek(string(key))

	return
}

// LSize returns the size of key in the bucket in the bucket at given bucket and key.
func (tx *Tx) LSize(bucket string, key []byte) (int, error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return 0, err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return 0, ErrBucket
	}
	if tx.CheckExpire(bucket, key) {
		return 0, ErrKeyNotFound
	}
	return l.Size(string(key))
}

// LRange returns the specified elements of the list stored in the bucket at given bucket,key, start and end.
// The offsets start and stop are zero-based indexes 0 being the first element of the list (the head of the list),
// 1 being the next element and so on.
// Start and end can also be negative numbers indicating offsets from the end of the list,
// where -1 is the last element of the list, -2 the penultimate element and so on.
func (tx *Tx) LRange(bucket string, key []byte, start, end int) (list [][]byte, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return nil, err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return nil, ErrBucket
	}
	if tx.CheckExpire(bucket, key) {
		return nil, ErrKeyNotFound
	}
	return l.LRange(string(key), start, end)
}

// LRem removes the first count occurrences of elements equal to value from the list stored in the bucket at given bucket,key,count.
// The count argument influences the operation in the following ways:
// count > 0: Remove elements equal to value moving from head to tail.
// count < 0: Remove elements equal to value moving from tail to head.
// count = 0: Remove all elements equal to value.
func (tx *Tx) LRem(bucket string, key []byte, count int, value []byte) (removedNum int, err error) {
	var (
		buffer bytes.Buffer
		size   int
	)
	size, err = tx.LSize(bucket, key)
	if err != nil {
		return 0, err
	}

	if count > size || count < -size {
		return 0, list.ErrCount
	}

	buffer.Write([]byte(strconv2.IntToStr(count)))
	buffer.Write([]byte(SeparatorForListKey))
	buffer.Write(value)
	newValue := buffer.Bytes()

	err = tx.push(bucket, key, DataLRemFlag, newValue)
	if err != nil {
		return 0, err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return 0, ErrBucket
	}

	removedNum, err = l.LRemNum(string(key), count, value)
	return
}

// LSet sets the list element at index to value.
func (tx *Tx) LSet(bucket string, key []byte, index int, value []byte) error {
	var (
		err    error
		buffer bytes.Buffer
	)

	if err = tx.checkTxIsClosed(); err != nil {
		return err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return ErrBucket
	}
	if tx.CheckExpire(bucket, key) {
		return ErrKeyNotFound
	}
	if _, ok := l.Items[string(key)]; !ok {
		return ErrKeyNotFound
	}

	size, _ := tx.LSize(bucket, key)
	if index < 0 || index >= size {
		return list.ErrIndexOutOfRange
	}

	buffer.Write(key)
	buffer.Write([]byte(SeparatorForListKey))
	indexBytes := []byte(strconv2.IntToStr(index))
	buffer.Write(indexBytes)
	newKey := buffer.Bytes()

	return tx.push(bucket, newKey, DataLSetFlag, value)
}

// LTrim trims an existing list so that it will contain only the specified range of elements specified.
// the offsets start and stop are zero-based indexes 0 being the first element of the list (the head of the list),
// 1 being the next element and so on.
// start and end can also be negative numbers indicating offsets from the end of the list,
// where -1 is the last element of the list, -2 the penultimate element and so on.
func (tx *Tx) LTrim(bucket string, key []byte, start, end int) error {
	var (
		err    error
		buffer bytes.Buffer
	)

	if err = tx.checkTxIsClosed(); err != nil {
		return err
	}

	l := tx.db.Index.getList(bucket)
	if l == nil {
		return ErrBucket
	}
	if tx.CheckExpire(bucket, key) {
		return ErrKeyNotFound
	}
	if _, ok := l.Items[string(key)]; !ok {
		return ErrKeyNotFound
	}

	if _, err := tx.LRange(bucket, key, start, end); err != nil {
		return err
	}

	buffer.Write(key)
	buffer.Write([]byte(SeparatorForListKey))
	buffer.Write([]byte(strconv2.IntToStr(start)))
	newKey := buffer.Bytes()

	return tx.push(bucket, newKey, DataLTrimFlag, []byte(strconv2.IntToStr(end)))
}

// LRemByIndex remove the list element at specified index
func (tx *Tx) LRemByIndex(bucket string, key []byte, indexes ...int) (removedNum int, err error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return 0, err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return 0, ErrBucket
	}
	if tx.CheckExpire(bucket, key) {
		return 0, ErrKeyNotFound
	}
	sort.Ints(indexes)
	removedNum, err = l.LRemByIndexPreCheck(string(key), indexes)
	if removedNum == 0 || err != nil {
		return
	}
	data, err := MarshalInts(indexes)
	if err != nil {
		return 0, err
	}
	err = tx.push(bucket, key, DataLRemByIndex, data)
	if err != nil {
		return 0, err
	}
	return
}

// LKeys find all keys matching a given pattern
func (tx *Tx) LKeys(bucket, pattern string, f func(key string) bool) error {
	if err := tx.checkTxIsClosed(); err != nil {
		return err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return ErrBucket
	}
	for key := range l.Items {
		if tx.CheckExpire(bucket, []byte(key)) {
			continue
		}
		if end, err := MatchForRange(pattern, key, f); end || err != nil {
			return err
		}
	}
	return nil
}

func (tx *Tx) ExpireList(bucket string, key []byte, ttl uint32) error {
	if err := tx.checkTxIsClosed(); err != nil {
		return err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return ErrBucket
	}
	l.TTL[string(key)] = ttl
	l.TimeStamp[string(key)] = uint64(time.Now().Unix())
	ttls := strconv2.Int64ToStr(int64(ttl))
	err := tx.push(bucket, key, DataExpireListFlag, []byte(ttls))
	if err != nil {
		return err
	}
	return nil
}

func (tx *Tx) CheckExpire(bucket string, key []byte) bool {
	l := tx.db.Index.getList(bucket)
	if l.IsExpire(string(key)) {
		_ = tx.push(bucket, key, DataDeleteFlag)
		return true
	}
	return false
}

func (tx *Tx) GetListTTL(bucket string, key []byte) (uint32, error) {
	if err := tx.checkTxIsClosed(); err != nil {
		return 0, err
	}
	l := tx.db.Index.getList(bucket)
	if l == nil {
		return 0, ErrBucket
	}
	return l.GetListTTL(string(key))
}
