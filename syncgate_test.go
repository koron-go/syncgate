package syncgate_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/koron-go/syncgate"
)

func TestWait1(t *testing.T) {
	var r []string

	ctx := context.Background()
	g := syncgate.New()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := syncgate.Wait(ctx, g)
		if err != nil {
			t.Errorf("syncgate.Wait failure: %s", err)
		}
		r = append(r, "done1")
	}()
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		r = append(r, "done2")
		g.Open()
	}()
	wg.Wait()

	if !reflect.DeepEqual([]string{"done2", "done1"}, r) {
		t.Fatalf("unexpected order: got=%+v", r)
	}
}

func TestWait2(t *testing.T) {
	var r1, r2 []string

	ctx := context.Background()
	g1 := syncgate.New()
	g2 := syncgate.New()

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		err := syncgate.Wait(ctx, g1, g2)
		if err != nil {
			t.Errorf("syncgate.Wait failure: %s", err)
		}
		r1 = append(r1, "done")
		r2 = append(r2, "done")
	}()
	time.Sleep(50 * time.Millisecond)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		r1 = append(r1, "done1")
		g1.Open()
	}()
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		r2 = append(r2, "done2")
		g2.Open()
	}()
	wg.Wait()

	if !reflect.DeepEqual([]string{"done1", "done"}, r1) {
		t.Fatalf("unexpected order of r1: got=%+v", r1)
	}
	if !reflect.DeepEqual([]string{"done2", "done"}, r2) {
		t.Fatalf("unexpected order of r2: got=%+v", r2)
	}
}

func TestCancel(t *testing.T) {
	var r1, r2 []string

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	g1 := syncgate.New()
	g2 := syncgate.New()

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		err := syncgate.Wait(ctx, g1, g2)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("unexpected failure: %s", err)
		}
	}()
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		r1 = append(r1, "done1")
		g1.Open()
	}()
	go func() {
		defer wg.Done()
		r2 = append(r2, "done2")
		g2.Open()
	}()
	wg.Wait()

	if !reflect.DeepEqual([]string{"done1"}, r1) {
		t.Fatalf("unexpected order: got=%+v", r1)
	}
	if !reflect.DeepEqual([]string{"done2"}, r2) {
		t.Fatalf("unexpected order: got=%+v", r2)
	}
}

func TestIsOpen(t *testing.T) {
	g := syncgate.New()
	if g.IsOpen() {
		t.Fatal("unexpected open")
	}
	g.Open()
	if !g.IsOpen() {
		t.Fatal("unexpected non-open")
	}
	// dual open.
	g.Open()
	if !g.IsOpen() {
		t.Fatal("unexpected non-open")
	}
}

func TestString(t *testing.T) {
	g := syncgate.New("common")
	if s := g.String(); s != "syncgate.Gate{ name=common opened=false }" {
		t.Fatalf("unexpected string: got=%s", s)
	}
	g.Open()
	if s := g.String(); s != "syncgate.Gate{ name=common opened=true }" {
		t.Fatalf("unexpected string: got=%s", s)
	}
}

func TestTry(t *testing.T) {
	g1 := syncgate.New()
	g2 := syncgate.New()
	if syncgate.Try(g1, g2) {
		t.Fatal("unexpected open")
	}
	g1.Open()
	if syncgate.Try(g1, g2) {
		t.Fatal("unexpected open")
	}
	g2.Open()
	if !syncgate.Try(g1, g2) {
		t.Fatal("unexpected non-open")
	}
	if !syncgate.Try() {
		t.Fatal("syncgate.Try with no gates should return true always")
	}
}

func TestWaitOpened(t *testing.T) {
	g := syncgate.New()
	g.Open()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := syncgate.Wait(ctx, g)
	if err != nil {
		t.Fatalf("failed to wait opened Gate: %s", err)
	}
}

func TestWait0(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := syncgate.Wait(ctx)
	if err != nil {
		t.Fatalf("syncgate.Wait without Gates should return soon: %s", err)
	}
}
