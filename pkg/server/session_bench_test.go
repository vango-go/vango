package server_test

import (
	"sync"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/server"
)

// BenchmarkSessionGet benchmarks Session.Get performance.
func BenchmarkSessionGet(b *testing.B) {
	session := server.NewMockSession()
	session.Set("key1", "value1")
	session.Set("key2", 12345)
	session.Set("key3", map[string]string{"foo": "bar"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.Get("key1")
	}
}

// BenchmarkSessionSet benchmarks Session.Set performance.
func BenchmarkSessionSet(b *testing.B) {
	session := server.NewMockSession()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Set("key", "value")
	}
}

// BenchmarkSessionGetParallel benchmarks concurrent Get operations.
func BenchmarkSessionGetParallel(b *testing.B) {
	session := server.NewMockSession()
	session.Set("key", "value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = session.Get("key")
		}
	})
}

// BenchmarkSessionSetParallel benchmarks concurrent Set operations.
func BenchmarkSessionSetParallel(b *testing.B) {
	session := server.NewMockSession()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			session.Set("key", i)
			i++
		}
	})
}

// BenchmarkSessionMixedParallel benchmarks concurrent Get/Set mix.
func BenchmarkSessionMixedParallel(b *testing.B) {
	session := server.NewMockSession()
	session.Set("key", "initial")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				_ = session.Get("key")
			} else {
				session.Set("key", i)
			}
			i++
		}
	})
}

// BenchmarkSessionHas benchmarks Session.Has performance.
func BenchmarkSessionHas(b *testing.B) {
	session := server.NewMockSession()
	session.Set("exists", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.Has("exists")
	}
}

// BenchmarkSessionDelete benchmarks Session.Delete performance.
func BenchmarkSessionDelete(b *testing.B) {
	session := server.NewMockSession()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Set("key", "value")
		session.Delete("key")
	}
}

// BenchmarkSessionGetString benchmarks type-safe getter.
func BenchmarkSessionGetString(b *testing.B) {
	session := server.NewMockSession()
	session.SetString("str", "hello world")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.GetString("str")
	}
}

// BenchmarkSessionGetInt benchmarks type-safe getter.
func BenchmarkSessionGetInt(b *testing.B) {
	session := server.NewMockSession()
	session.SetInt("num", 42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.GetInt("num")
	}
}

// TestSessionConcurrentAccess tests thread safety under contention.
func TestSessionConcurrentAccess(t *testing.T) {
	session := server.NewMockSession()

	var wg sync.WaitGroup
	goroutines := 100
	iterations := 1000

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := "key"
				if i%3 == 0 {
					session.Set(key, i)
				} else if i%3 == 1 {
					_ = session.Get(key)
				} else {
					_ = session.Has(key)
				}
			}
		}(g)
	}

	wg.Wait()
	// No panic = success
}
