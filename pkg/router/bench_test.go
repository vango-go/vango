package router

import (
	"fmt"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/server"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// BenchmarkRouterMatchStatic benchmarks matching a static route.
func BenchmarkRouterMatchStatic(b *testing.B) {
	r := NewRouter()

	// Add some routes
	paths := []string{"/", "/about", "/contact", "/pricing", "/features"}
	for _, p := range paths {
		r.AddPage(p, func(ctx server.Ctx, params any) vdom.Component {
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/about")
	}
}

// BenchmarkRouterMatchParam benchmarks matching a parameterized route.
func BenchmarkRouterMatchParam(b *testing.B) {
	r := NewRouter()
	r.AddPage("/users/:id", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/users/123")
	}
}

// BenchmarkRouterMatchMultipleParams benchmarks matching multiple parameters.
func BenchmarkRouterMatchMultipleParams(b *testing.B) {
	r := NewRouter()
	r.AddPage("/users/:userId/posts/:postId/comments/:commentId", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/users/42/posts/100/comments/999")
	}
}

// BenchmarkRouterMatchCatchAll benchmarks matching a catch-all route.
func BenchmarkRouterMatchCatchAll(b *testing.B) {
	r := NewRouter()
	r.AddPage("/files/*path", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/files/a/b/c/d/e")
	}
}

// BenchmarkRouterMatchDeep benchmarks matching in a deep tree.
func BenchmarkRouterMatchDeep(b *testing.B) {
	r := NewRouter()
	r.AddPage("/a/b/c/d/e/f/g/h", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/a/b/c/d/e/f/g/h")
	}
}

// BenchmarkRouterMatchLargeTree benchmarks matching in a large tree.
func BenchmarkRouterMatchLargeTree(b *testing.B) {
	r := NewRouter()

	// Add 100 routes
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/route%d", i)
		r.AddPage(path, func(ctx server.Ctx, params any) vdom.Component {
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/route50")
	}
}

// BenchmarkRouterMatchNoMatch benchmarks failed matches.
func BenchmarkRouterMatchNoMatch(b *testing.B) {
	r := NewRouter()
	r.AddPage("/users", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match("GET", "/notfound")
	}
}

// BenchmarkParamParse benchmarks parameter parsing.
func BenchmarkParamParse(b *testing.B) {
	type Params struct {
		ID   int    `param:"id"`
		Name string `param:"name"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "123", "name": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p Params
		parser.Parse(params, &p)
	}
}

// BenchmarkSplitPath benchmarks path splitting.
func BenchmarkSplitPath(b *testing.B) {
	path := "/users/123/posts/456/comments"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitPath(path)
	}
}
