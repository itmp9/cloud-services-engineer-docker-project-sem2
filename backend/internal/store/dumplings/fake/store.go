package fake

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"gitlab.praktikum-services.ru/Stasyan/momo-store/internal/store/dumplings"
)

// Store is a fake in-memory implementation of dumplings.Store
type Store struct {
	rand              *rand.Rand
	orderID           int64
	orderIDPath       string
	orderIDKey        []byte
	availableProducts []dumplings.Product
}

func NewStore() *Store {
	return &Store{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func NewPersistentStore(orderIDPath string, orderIDKey []byte) *Store {
	return &Store{
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		orderIDPath: orderIDPath,
		orderIDKey:  append([]byte(nil), orderIDKey...),
	}
}

func (s *Store) SetAvailablePacks(products ...dumplings.Product) {
	s.availableProducts = products
}

func (s *Store) ListProducts(_ context.Context) ([]dumplings.Product, error) {
	return s.availableProducts, nil
}

func (s *Store) CreateOrder(_ context.Context, _ ...dumplings.OrderItem) (id int64, err error) {
	if s.orderIDPath != "" {
		return s.nextPersistentOrderID()
	}

	return atomic.AddInt64(&s.orderID, 1), nil
}

func (s *Store) nextPersistentOrderID() (int64, error) {
	file, err := os.OpenFile(s.orderIDPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return 0, err
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, err
	}

	var sequence int64
	if value := strings.TrimSpace(string(data)); value != "" {
		sequence, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid order sequence: %w", err)
		}
	}
	sequence++

	if err := file.Truncate(0); err != nil {
		return 0, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	if _, err := fmt.Fprintln(file, sequence); err != nil {
		return 0, err
	}
	if err := file.Sync(); err != nil {
		return 0, err
	}

	var input [8]byte
	binary.BigEndian.PutUint64(input[:], uint64(sequence))
	mac := hmac.New(sha256.New, s.orderIDKey)
	_, _ = mac.Write(input[:])
	id := int64(binary.BigEndian.Uint64(mac.Sum(nil)[:8]) & uint64(^uint64(0)>>1))
	if id == 0 {
		return sequence, nil
	}

	return id, nil
}
