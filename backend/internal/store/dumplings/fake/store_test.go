package fake

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestPersistentOrderIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.seq")
	key := []byte("01234567890123456789012345678901")
	stores := []*Store{
		NewPersistentStore(path, key),
		NewPersistentStore(path, key),
	}

	const count = 50
	ids := make(chan int64, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup

	for index := 0; index < count; index++ {
		wg.Add(1)
		go func(store *Store) {
			defer wg.Done()
			id, err := store.CreateOrder(context.Background())
			if err != nil {
				errs <- err
				return
			}
			ids <- id
		}(stores[index%len(stores)])
	}

	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}

	unique := make(map[int64]struct{}, count)
	for id := range ids {
		unique[id] = struct{}{}
	}
	if len(unique) != count {
		t.Fatalf("expected %d unique IDs, got %d", count, len(unique))
	}

	assertSequence(t, path, count)

	if _, err := NewPersistentStore(path, key).CreateOrder(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertSequence(t, path, count+1)
}

func assertSequence(t *testing.T, path string, expected int) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	if value != expected {
		t.Fatalf("expected sequence %d, got %d", expected, value)
	}
}
