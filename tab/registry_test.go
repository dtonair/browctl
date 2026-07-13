package tab

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dt/browctl/protocol"
)

type fakeProvider struct{}

func (fakeProvider) BrowserContext(profile string) (context.Context, error) {
	return context.Background(), nil
}

func TestRegisterReturnsStableIDForSameTarget(t *testing.T) {
	r := NewRegistry(fakeProvider{})
	r.idMaker = func() string { return "01STABLESTABLESTABLESTAB" }

	first := r.Register("work", "target-1", "https://example.com", "Example")
	second := r.Register("work", "target-1", "https://example.org", "Changed")

	if first.ID != second.ID {
		t.Fatalf("IDs differ: %s vs %s", first.ID, second.ID)
	}
	if second.URL != "https://example.org" || second.Title != "Changed" {
		t.Fatalf("second = %#v, want updated metadata", second)
	}
}

func TestLookupActiveAndMissing(t *testing.T) {
	r := NewRegistry(fakeProvider{})
	r.idMaker = func() string { return "01ACTIVEACTIVEACTIVEACT" }
	tab := r.Register("work", "target-1", "https://example.com", "")

	got, err := r.lookup("work", "active")
	if err != nil {
		t.Fatalf("lookup active error = %v", err)
	}
	if got.ID != tab.ID {
		t.Fatalf("active ID = %s, want %s", got.ID, tab.ID)
	}

	_, err = r.lookup("work", "missing")
	if err == nil {
		t.Fatal("lookup missing err = nil, want TAB_NOT_FOUND")
	}
	if perr := err.(*protocol.Error); perr.Code != protocol.TabNotFound {
		t.Fatalf("code = %s, want TAB_NOT_FOUND", perr.Code)
	}
}

func TestWithTabSerializesPerTab(t *testing.T) {
	r := NewRegistry(fakeProvider{})
	r.idMaker = func() string { return "01SERIALSERIALSERIALSE" }
	tab := r.Register("work", "target-1", "", "")

	const n = 25
	order := make([]int, 0, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := r.WithTab(context.Background(), "work", tab.ID, func(ctx context.Context, tab Tab) error {
				currentLen := len(order)
				time.Sleep(time.Millisecond)
				if len(order) != currentLen {
					t.Errorf("critical section interleaved: len changed from %d to %d", currentLen, len(order))
				}
				order = append(order, i)
				return nil
			})
			if err != nil {
				t.Errorf("WithTab() error = %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(order) != n {
		t.Fatalf("len(order) = %d, want %d", len(order), n)
	}
}

func TestRemoveChoosesNextActiveTab(t *testing.T) {
	r := NewRegistry(fakeProvider{})
	ids := []string{"01FIRSTFIRSTFIRSTFIRSTFI", "01SECONDCSECONDCSECOND"}
	r.idMaker = func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}
	first := r.Register("work", "target-1", "", "")
	second := r.Register("work", "target-2", "", "")

	r.remove("work", first.ID, first.TargetID)
	active, err := r.lookup("work", "active")
	if err != nil {
		t.Fatalf("lookup active error = %v", err)
	}
	if active.ID != second.ID {
		t.Fatalf("active = %s, want %s", active.ID, second.ID)
	}
}
