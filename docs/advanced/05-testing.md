# Testing

## Unit Testing Components

```go
func TestCounter(t *testing.T) {
    ctx := vango.TestContext()
    c := Counter(5)
    tree := ctx.Mount(c)

    assert.Contains(t, tree.Text(), "Count: 5")

    ctx.Click("[data-testid=increment]")
    assert.Contains(t, tree.Text(), "Count: 6")
}
```

## Testing Signals

```go
func TestSignal(t *testing.T) {
    ctx := vango.TestContext()
    count := ctx.Signal(0)

    count.Set(10)
    ctx.Flush()

    assert.Equal(t, 10, count())
}
```

## Integration Testing

```go
func TestLoginFlow(t *testing.T) {
    app := vango.TestApp()
    page := app.Navigate("/login")

    page.Fill("[name=email]", "user@example.com")
    page.Fill("[name=password]", "secret")
    page.Click("[type=submit]")

    assert.Equal(t, "/dashboard", page.URL())
}
```

## E2E Testing

Use Playwright or Cypress — Vango renders standard HTML:

```typescript
test('user can log in', async ({ page }) => {
    await page.goto('/login');
    await page.fill('[name=email]', 'test@example.com');
    await page.click('[type=submit]');
    await expect(page).toHaveURL('/dashboard');
});
```
