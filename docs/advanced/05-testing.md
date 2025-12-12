# Testing

Vango provides testing helpers for fast, reliable component tests.

## Using vtest

```go
import "github.com/vango-dev/vango/v2/pkg/vtest"

func TestDashboard_Authenticated(t *testing.T) {
    // Create context with authenticated user
    ctx := vtest.NewCtx().
        WithUser(&User{ID: "123", Role: "admin"}).
        Build()

    // Render component
    comp, err := Dashboard(ctx)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Assert content
    vtest.ExpectContains(t, comp, "Welcome")
    vtest.ExpectNotContains(t, comp, "Login")
}
```

## Context Builder

```go
// Empty context (unauthenticated)
ctx := vtest.NewCtx().Build()

// With authenticated user
ctx := vtest.NewCtx().
    WithUser(&models.User{ID: "123", Email: "test@example.com"}).
    Build()

// With session data
ctx := vtest.NewCtx().
    WithUser(user).
    WithData("theme", "dark").
    WithData("locale", "en").
    Build()

// With URL parameters
ctx := vtest.NewCtx().
    WithParam("id", "123").
    Build()

// Shorthand for WithUser
ctx := vtest.CtxWithUser(&models.User{Role: "admin"})
```

## Render Assertions

```go
// Contains text
vtest.ExpectContains(t, comp, "Welcome")

// Does not contain text
vtest.ExpectNotContains(t, comp, "Error")

// Get HTML string for custom assertions
html := vtest.RenderToString(comp)
assert.Regexp(t, `class="active"`, html)
```

## Testing Auth Guards

```go
func TestDashboard_Unauthenticated(t *testing.T) {
    ctx := vtest.NewCtx().Build()  // No user

    _, err := Dashboard(ctx)
    
    if err != auth.ErrUnauthorized {
        t.Errorf("expected ErrUnauthorized, got %v", err)
    }
}

func TestAdminPage_WrongRole(t *testing.T) {
    ctx := vtest.NewCtx().
        WithUser(&User{Role: "member"}).
        Build()

    _, err := AdminPage(ctx)
    
    if err != auth.ErrForbidden {
        t.Errorf("expected ErrForbidden, got %v", err)
    }
}
```

## Testing Signals

```go
func TestCounter(t *testing.T) {
    ctx := vtest.NewCtx().Build()
    
    comp := Counter(5)
    html := vtest.RenderToString(comp)
    
    if !strings.Contains(html, "Count: 5") {
        t.Error("expected initial count")
    }
}
```

## Testing Toast Emissions

```go
type mockEmitter struct {
    events []any
}

func (m *mockEmitter) Emit(name string, data any) {
    m.events = append(m.events, data)
}

func TestSaveEmitsToast(t *testing.T) {
    emitter := &mockEmitter{}
    ctx := vtest.NewCtx().WithEmitter(emitter).Build()
    
    SaveProject(ctx, input)
    
    if len(emitter.events) != 1 {
        t.Error("expected toast to be emitted")
    }
}
```

## E2E Testing

For end-to-end tests, use Playwright or Cypress—Vango renders standard HTML:

```typescript
import { test, expect } from '@playwright/test';

test('user can create project', async ({ page }) => {
    await page.goto('/projects/new');
    await page.fill('[name=name]', 'My Project');
    await page.click('[type=submit]');
    
    await expect(page).toHaveURL(/\/projects\/\d+/);
    await expect(page.locator('h1')).toContainText('My Project');
});
```

## Best Practices

1. **Test behavior, not implementation** - Focus on what user sees
2. **Use vtest.NewCtx()** - Don't create mock sessions manually
3. **Test error cases** - Verify auth guards work
4. **Keep tests fast** - No database in unit tests
