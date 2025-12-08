# Phase 8: Higher-Level Features ✅ COMPLETE

> **The APIs that make Vango productive for real applications**

**Status**: Complete (2024-12-07)

---

## Overview

Phase 8 builds the higher-level abstractions that developers interact with daily. These features sit on top of the foundation built in Phases 1-7 and provide ergonomic APIs for common patterns.

**Implementation**: All subsystems implemented in `pkg/features/` with >80% test coverage.

### Subsystems

| Subsystem | Purpose | Priority | Status |
|-----------|---------|----------|--------|
| 8.1 Forms & Validation | Structured form handling with validation | High | ✅ Complete |
| 8.2 Resources | Async data loading with states | High | ✅ Complete |
| 8.3 Context API | Dependency injection through component tree | High | ✅ Complete |
| 8.4 URL State | Sync component state with URL | Medium | ✅ Complete |
| 8.5 Client Hooks | 60fps interactions with server state | High | ✅ Complete |
| 8.6 Shared State | Session and global signals | High | ✅ Complete |
| 8.7 Optimistic Updates | Instant feedback for interactions | Medium | ✅ Complete |
| 8.8 JavaScript Islands | Third-party library integration | Medium | ✅ Complete |

---

## 8.1 Forms & Validation

### Goals

1. Type-safe form binding to Go structs
2. Declarative validation with built-in validators
3. Field-level error display
4. Support for nested objects and arrays
5. Async validation (e.g., username availability)

### Public API

#### UseForm Hook

```go
// Define form structure with validation tags
type ContactForm struct {
    Name    string `form:"name" validate:"required,min=2,max=100"`
    Email   string `form:"email" validate:"required,email"`
    Phone   string `form:"phone" validate:"phone"`
    Message string `form:"message" validate:"required,max=1000"`
}

func ContactPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(ContactForm{})

        submit := func() {
            if !form.Validate() {
                return // Errors displayed automatically
            }

            // Get typed values
            data := form.Values()
            sendEmail(data.Name, data.Email, data.Message)

            form.Reset()
        }

        return Form(OnSubmit(submit),
            // form.Field wraps input with error display
            form.Field("Name",
                Input(Type("text"), Placeholder("Your name")),
            ),
            form.Field("Email",
                Input(Type("email"), Placeholder("you@example.com")),
            ),
            form.Field("Phone",
                Input(Type("tel"), Placeholder("Optional")),
            ),
            form.Field("Message",
                Textarea(Rows(5), Placeholder("Your message")),
            ),

            Button(
                Type("submit"),
                Disabled(form.IsSubmitting()),
                Text("Send"),
            ),
        )
    })
}
```

#### Form Type

```go
type Form[T any] struct {
    // Private fields
}

// Constructor
func UseForm[T any](initial T) *Form[T]

// Read values
func (f *Form[T]) Values() T                    // Get struct with current values
func (f *Form[T]) Get(field string) any         // Get single field value
func (f *Form[T]) GetString(field string) string
func (f *Form[T]) GetInt(field string) int
func (f *Form[T]) GetBool(field string) bool

// Write values
func (f *Form[T]) Set(field string, value any)  // Set single field
func (f *Form[T]) SetValues(values T)           // Set all values
func (f *Form[T]) Reset()                       // Reset to initial values

// Validation
func (f *Form[T]) Validate() bool               // Run all validators, return success
func (f *Form[T]) ValidateField(field string) bool
func (f *Form[T]) Errors() map[string][]string  // All errors by field
func (f *Form[T]) FieldErrors(field string) []string
func (f *Form[T]) HasError(field string) bool
func (f *Form[T]) ClearErrors()
func (f *Form[T]) SetError(field string, msg string)

// State
func (f *Form[T]) IsDirty() bool                // Any field changed
func (f *Form[T]) FieldDirty(field string) bool
func (f *Form[T]) IsSubmitting() bool
func (f *Form[T]) SetSubmitting(bool)
func (f *Form[T]) IsValid() bool                // No validation errors

// Field binding (creates wrapped input with error display)
func (f *Form[T]) Field(name string, input *vango.VNode, validators ...Validator) *vango.VNode
```

#### Built-in Validators

```go
// String validators
Required(msg string) Validator           // Non-empty
MinLength(n int, msg string) Validator   // len >= n
MaxLength(n int, msg string) Validator   // len <= n
Pattern(regex string, msg string) Validator
Email(msg string) Validator
URL(msg string) Validator
UUID(msg string) Validator
Alpha(msg string) Validator              // Letters only
AlphaNumeric(msg string) Validator
Numeric(msg string) Validator            // Digits only

// Numeric validators
Min(n any, msg string) Validator         // >= n
Max(n any, msg string) Validator         // <= n
Between(min, max any, msg string) Validator
Positive(msg string) Validator           // > 0
NonNegative(msg string) Validator        // >= 0

// Comparison validators
EqualTo(field string, msg string) Validator      // Must match another field
NotEqualTo(field string, msg string) Validator

// Date validators
DateAfter(t time.Time, msg string) Validator
DateBefore(t time.Time, msg string) Validator
Future(msg string) Validator
Past(msg string) Validator

// Custom validator
Custom(fn func(value any) error) Validator

// Async validator (for server-side checks)
Async(fn func(value any) (error, bool)) Validator
// Returns (error, isComplete) - isComplete=false while loading
```

#### Field Method Implementation

```go
func (f *Form[T]) Field(name string, input *vango.VNode, validators ...Validator) *vango.VNode {
    // Get current value
    value := f.Get(name)

    // Get errors
    errors := f.FieldErrors(name)
    hasError := len(errors) > 0

    // Clone input with value binding
    boundInput := cloneWithProps(input, vdom.Props{
        "value": value,
        "name":  name,
        "class": conditionalClass(input.Props["class"], "field-error", hasError),
    })

    // Add OnInput handler for binding
    boundInput = addHandler(boundInput, "onInput", func(newValue string) {
        f.Set(name, newValue)
    })

    // Add OnBlur for validation
    boundInput = addHandler(boundInput, "onBlur", func() {
        f.ValidateField(name)
    })

    // Wrap with error display
    return Div(Class("field"),
        boundInput,
        If(hasError,
            Div(Class("field-errors"),
                Range(errors, func(err string, i int) *vango.VNode {
                    return Span(Class("field-error-message"), Text(err))
                }),
            ),
        ),
    )
}
```

### Form Arrays

```go
type OrderForm struct {
    CustomerName string      `form:"customer_name" validate:"required"`
    Items        []OrderItem `form:"items" validate:"min=1"`
}

type OrderItem struct {
    ProductID int `form:"product_id" validate:"required"`
    Quantity  int `form:"quantity" validate:"required,min=1"`
}

func OrderPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(OrderForm{})

        addItem := func() {
            form.AppendTo("Items", OrderItem{})
        }

        return Form(
            form.Field("CustomerName", Input(Type("text"))),

            H3(Text("Items")),

            // Render array items
            form.Array("Items", func(item FormArrayItem, i int) *vango.VNode {
                return Div(Class("item-row"), Key(i),
                    item.Field("ProductID",
                        Select(
                            Option(Value(""), Text("Select product")),
                            Option(Value("1"), Text("Widget")),
                            Option(Value("2"), Text("Gadget")),
                        ),
                    ),
                    item.Field("Quantity",
                        Input(Type("number"), Min("1")),
                    ),
                    Button(
                        Type("button"),
                        OnClick(item.Remove),
                        Text("Remove"),
                    ),
                )
            }),

            Button(Type("button"), OnClick(addItem), Text("Add Item")),
            Button(Type("submit"), Text("Place Order")),
        )
    })
}
```

#### FormArrayItem Type

```go
type FormArrayItem struct {
    form  *Form[any]
    path  string  // e.g., "Items.0"
    index int
}

func (i FormArrayItem) Field(name string, input *vango.VNode, validators ...Validator) *vango.VNode {
    fullPath := i.path + "." + name
    return i.form.Field(fullPath, input, validators...)
}

func (i FormArrayItem) Remove() {
    i.form.RemoveAt(i.path[:strings.LastIndex(i.path, ".")], i.index)
}

func (i FormArrayItem) Index() int {
    return i.index
}
```

### Internal Implementation

```go
type Form[T any] struct {
    initial    T
    values     *Signal[T]
    errors     *Signal[map[string][]string]
    touched    *Signal[map[string]bool]
    submitting *Signal[bool]
    validators map[string][]Validator
    fieldMeta  map[string]fieldMeta
}

type fieldMeta struct {
    formTag     string
    validateTag string
    fieldType   reflect.Type
}

func UseForm[T any](initial T) *Form[T] {
    f := &Form[T]{
        initial:    initial,
        values:     Signal(initial),
        errors:     Signal(make(map[string][]string)),
        touched:    Signal(make(map[string]bool)),
        submitting: Signal(false),
        validators: make(map[string][]Validator),
        fieldMeta:  make(map[string]fieldMeta),
    }

    // Parse struct tags
    f.parseStructTags(reflect.TypeOf(initial))

    return f
}

func (f *Form[T]) parseStructTags(t reflect.Type) {
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)

        formTag := field.Tag.Get("form")
        if formTag == "" {
            formTag = strings.ToLower(field.Name)
        }

        validateTag := field.Tag.Get("validate")

        f.fieldMeta[formTag] = fieldMeta{
            formTag:     formTag,
            validateTag: validateTag,
            fieldType:   field.Type,
        }

        // Parse validate tag into validators
        if validateTag != "" {
            f.validators[formTag] = parseValidateTag(validateTag)
        }

        // Recurse for nested structs
        if field.Type.Kind() == reflect.Struct {
            f.parseStructTags(field.Type)
        }
    }
}

func (f *Form[T]) Validate() bool {
    allErrors := make(map[string][]string)

    for field, validators := range f.validators {
        value := f.Get(field)
        var fieldErrors []string

        for _, v := range validators {
            if err := v.Validate(value); err != nil {
                fieldErrors = append(fieldErrors, err.Error())
            }
        }

        if len(fieldErrors) > 0 {
            allErrors[field] = fieldErrors
        }
    }

    f.errors.Set(allErrors)
    return len(allErrors) == 0
}
```

---

## 8.2 Resources

### Goals

1. Handle async data loading with loading/error/success states
2. Automatic dependency tracking (re-fetch when deps change)
3. Built-in caching with stale time
4. Manual refetch capability
5. Optimistic mutations

### Public API

```go
// Basic resource
func Resource[T any](fetcher func() (T, error)) *Resource[T]

// Resource with key (re-fetches when key changes)
func ResourceWithKey[K comparable, T any](key func() K, fetcher func(K) (T, error)) *Resource[T]

// Resource type
type Resource[T any] struct {
    // Private
}

// State
type ResourceState int
const (
    ResourcePending ResourceState = iota  // Not started
    ResourceLoading                       // In progress
    ResourceReady                         // Success
    ResourceError                         // Failed
)

// State access
func (r *Resource[T]) State() ResourceState
func (r *Resource[T]) IsLoading() bool
func (r *Resource[T]) IsReady() bool
func (r *Resource[T]) IsError() bool

// Data access
func (r *Resource[T]) Data() T              // Returns zero value if not ready
func (r *Resource[T]) DataOr(fallback T) T  // Returns fallback if not ready
func (r *Resource[T]) Error() error         // Returns nil if no error

// Control
func (r *Resource[T]) Refetch()             // Trigger manual refetch
func (r *Resource[T]) Invalidate()          // Mark stale, refetch on next read
func (r *Resource[T]) Mutate(fn func(T) T)  // Optimistic local update

// Pattern matching
func (r *Resource[T]) Match(handlers ...ResourceHandler[T]) *vango.VNode
```

### Usage Examples

```go
// Basic usage
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        user := Resource(func() (*User, error) {
            return db.Users.FindByID(userID)
        })

        // Pattern 1: Switch on state
        switch user.State() {
        case ResourcePending, ResourceLoading:
            return LoadingSpinner()
        case ResourceError:
            return ErrorMessage(user.Error())
        case ResourceReady:
            return UserCard(user.Data())
        }
        return nil
    })
}

// Pattern 2: Match helper
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        user := Resource(func() (*User, error) {
            return db.Users.FindByID(userID)
        })

        return user.Match(
            OnLoading(func() *vango.VNode {
                return LoadingSpinner()
            }),
            OnError(func(err error) *vango.VNode {
                return ErrorMessage(err)
            }),
            OnReady(func(u *User) *vango.VNode {
                return UserCard(u)
            }),
        )
    })
}

// With reactive key (refetches when userID signal changes)
func UserProfile() vango.Component {
    return vango.Func(func() *vango.VNode {
        userID := UseURLState("id", 0)

        user := ResourceWithKey(
            func() int { return userID() },
            func(id int) (*User, error) {
                return db.Users.FindByID(id)
            },
        )

        // Automatically refetches when userID changes
        return user.Match(...)
    })
}

// With options
func ProjectList() vango.Component {
    return vango.Func(func() *vango.VNode {
        projects := Resource(func() ([]Project, error) {
            return db.Projects.All()
        }).
            StaleTime(5 * time.Minute).      // Cache for 5 minutes
            RetryOnError(3, time.Second).    // Retry 3 times
            OnSuccess(func(p []Project) {
                analytics.TrackLoad("projects", len(p))
            }).
            OnError(func(err error) {
                logger.Error("Failed to load projects", "error", err)
            })

        return projects.Match(...)
    })
}

// Manual refetch with button
func DataDashboard() vango.Component {
    return vango.Func(func() *vango.VNode {
        stats := Resource(fetchStats)

        return Div(
            Button(
                OnClick(stats.Refetch),
                Disabled(stats.IsLoading()),
                Text("Refresh"),
            ),
            stats.Match(...),
        )
    })
}
```

### Match Handlers

```go
type ResourceHandler[T any] interface {
    handle(*Resource[T]) *vango.VNode
}

// Handler constructors
func OnPending[T any](fn func() *vango.VNode) ResourceHandler[T]
func OnLoading[T any](fn func() *vango.VNode) ResourceHandler[T]
func OnError[T any](fn func(error) *vango.VNode) ResourceHandler[T]
func OnReady[T any](fn func(T) *vango.VNode) ResourceHandler[T]

// Convenience: combined pending/loading
func OnLoadingOrPending[T any](fn func() *vango.VNode) ResourceHandler[T]

// Implementation
func (r *Resource[T]) Match(handlers ...ResourceHandler[T]) *vango.VNode {
    for _, h := range handlers {
        if result := h.handle(r); result != nil {
            return result
        }
    }
    return nil
}
```

### Internal Implementation

```go
type Resource[T any] struct {
    fetcher     func() (T, error)
    state       *Signal[ResourceState]
    data        *Signal[T]
    error       *Signal[error]
    staleTime   time.Duration
    lastFetch   time.Time
    retryCount  int
    retryDelay  time.Duration
    onSuccess   func(T)
    onError     func(error)
    owner       *Owner
}

func Resource[T any](fetcher func() (T, error)) *Resource[T] {
    r := &Resource[T]{
        fetcher:   fetcher,
        state:     Signal(ResourcePending),
        data:      Signal(*new(T)),
        error:     Signal[error](nil),
        staleTime: 0, // No caching by default
        owner:     getCurrentOwner(),
    }

    // Start fetch on creation
    r.fetch()

    return r
}

func (r *Resource[T]) fetch() {
    r.state.Set(ResourceLoading)
    r.error.Set(nil)

    go func() {
        data, err := r.fetcher()

        if err != nil {
            r.error.Set(err)
            r.state.Set(ResourceError)

            if r.onError != nil {
                r.onError(err)
            }

            // Retry logic
            if r.retryCount > 0 {
                r.retryCount--
                time.Sleep(r.retryDelay)
                r.fetch()
            }
            return
        }

        r.data.Set(data)
        r.state.Set(ResourceReady)
        r.lastFetch = time.Now()

        if r.onSuccess != nil {
            r.onSuccess(data)
        }
    }()
}

func (r *Resource[T]) Refetch() {
    r.fetch()
}

func (r *Resource[T]) Data() T {
    // Check if stale
    if r.staleTime > 0 && time.Since(r.lastFetch) > r.staleTime {
        r.Refetch()
    }
    return r.data()
}

func (r *Resource[T]) Mutate(fn func(T) T) {
    // Optimistic update
    current := r.data()
    r.data.Set(fn(current))
}
```

---

## 8.3 Context API

### Goals

1. Pass values through component tree without prop drilling
2. Type-safe with generics
3. Default values when no provider
4. Multiple contexts can be nested

### Public API

```go
// Create a context with default value
func CreateContext[T any](defaultValue T) *Context[T]

// Context type
type Context[T any] struct {
    // Private
}

// Provide value to children
func (c *Context[T]) Provider(value T, children ...any) *vango.VNode

// Consume value (must be called within Provider subtree)
func (c *Context[T]) Use() T
```

### Usage Examples

```go
// Define contexts (package level)
var ThemeContext = vango.CreateContext("light")
var UserContext = vango.CreateContext[*User](nil)
var LocaleContext = vango.CreateContext("en")

// Provide at app level
func App() vango.Component {
    return vango.Func(func() *vango.VNode {
        theme := Signal("dark")
        user := getCurrentUser()
        locale := getUserLocale()

        return ThemeContext.Provider(theme(),
            UserContext.Provider(user,
                LocaleContext.Provider(locale,
                    Router(),
                ),
            ),
        )
    })
}

// Consume anywhere in the tree
func ThemedButton(text string) *vango.VNode {
    theme := ThemeContext.Use()

    return Button(
        Class("btn", theme+"-theme"),
        Text(text),
    )
}

func UserMenu() *vango.VNode {
    user := UserContext.Use()

    if user == nil {
        return LoginButton()
    }

    return Div(Class("user-menu"),
        Img(Src(user.AvatarURL)),
        Text(user.Name),
    )
}

func LocalizedText(key string) *vango.VNode {
    locale := LocaleContext.Use()
    text := i18n.Translate(key, locale)
    return Text(text)
}
```

### Internal Implementation

```go
type Context[T any] struct {
    id           uint64
    defaultValue T
}

var contextIDCounter uint64

func CreateContext[T any](defaultValue T) *Context[T] {
    return &Context[T]{
        id:           atomic.AddUint64(&contextIDCounter, 1),
        defaultValue: defaultValue,
    }
}

// Context values are stored on the Owner (component scope)
func (c *Context[T]) Provider(value T, children ...any) *vango.VNode {
    // Create a fragment that sets context in owner
    return Fragment(
        ContextSetter(c.id, value),
        children,
    )
}

func (c *Context[T]) Use() T {
    owner := getCurrentOwner()

    // Walk up owner chain looking for context value
    for o := owner; o != nil; o = o.parent {
        if val, ok := o.contexts[c.id]; ok {
            return val.(T)
        }
    }

    // Return default if not found
    return c.defaultValue
}

// Special VNode that sets context on owner during render
type contextSetterNode struct {
    contextID uint64
    value     any
}

func ContextSetter(id uint64, value any) *vango.VNode {
    // This is a special node that the renderer handles
    return &vango.VNode{
        Kind: vdom.KindContext,
        Props: vdom.Props{
            "contextID": id,
            "value":     value,
        },
    }
}
```

---

## 8.4 URL State

### Goals

1. Sync component state with URL query parameters
2. Browser back/forward navigation works
3. Shareable URLs
4. Type coercion (string, int, bool, arrays)
5. Debounce option for rapid changes

### Public API

```go
// Create URL state bound to query parameter
func UseURLState[T any](key string, defaultValue T) *URLState[T]

// URLState type
type URLState[T any] struct {
    // Private
}

// Read current value
func (u *URLState[T]) Get() T
func (u *URLState[T]) Call() T  // Enables u() syntax

// Write value (pushes history)
func (u *URLState[T]) Set(value T)

// Write value (replaces history, no back navigation)
func (u *URLState[T]) Replace(value T)

// Reset to default
func (u *URLState[T]) Reset()

// Check if non-default value
func (u *URLState[T]) IsSet() bool

// Options
func (u *URLState[T]) Debounce(duration time.Duration) *URLState[T]
func (u *URLState[T]) Serialize(fn func(T) string) *URLState[T]
func (u *URLState[T]) Deserialize(fn func(string) T) *URLState[T]
```

### Usage Examples

```go
func ProductList() vango.Component {
    return vango.Func(func() *vango.VNode {
        // URL: /products?search=laptop&page=2&sort=price
        search := UseURLState("search", "")
        page := UseURLState("page", 1)
        sort := UseURLState("sort", "newest")

        // Debounce search to avoid rapid URL updates
        search = search.Debounce(300 * time.Millisecond)

        products := Resource(func() ([]Product, error) {
            return db.Products.Search(search(), page(), sort())
        })

        return Div(
            // Search updates URL
            Input(
                Type("search"),
                Value(search()),
                OnInput(func(v string) { search.Set(v) }),
                Placeholder("Search products..."),
            ),

            // Sort dropdown updates URL
            Select(
                Value(sort()),
                OnChange(func(v string) { sort.Set(v) }),
                Option(Value("newest"), Text("Newest")),
                Option(Value("price_asc"), Text("Price: Low to High")),
                Option(Value("price_desc"), Text("Price: High to Low")),
            ),

            // Product grid
            ProductGrid(products),

            // Pagination (updates URL)
            Pagination(page(), totalPages, func(p int) { page.Set(p) }),
        )
    })
}

// Array values
func FilteredList() vango.Component {
    return vango.Func(func() *vango.VNode {
        // URL: /items?tags=go&tags=web&tags=framework
        tags := UseURLState("tags", []string{})

        return Div(
            TagSelector(tags(), func(t []string) { tags.Set(t) }),
            ItemList(tags()),
        )
    })
}
```

### Hash State (for modals, tabs)

```go
// Sync with URL hash instead of query params
func UseHashState(defaultValue string) *HashState

func TabPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        // URL: /settings#billing
        activeTab := UseHashState("general")

        return Div(
            TabList(
                Tab("general", "General", activeTab),
                Tab("billing", "Billing", activeTab),
                Tab("security", "Security", activeTab),
            ),
            TabPanel(activeTab()),
        )
    })
}
```

### Navigation with State

```go
// Navigate with URL params
vango.Navigate("/products", vango.WithParams(map[string]any{
    "search": "laptop",
    "page":   1,
    "sort":   "price_asc",
}))
// Results in: /products?search=laptop&page=1&sort=price_asc

// Get current URL params
params := vango.URLParams()
search := params.Get("search")

// Preserve params during navigation
vango.Navigate("/products/123", vango.PreserveParams("search", "sort"))
```

### Internal Implementation

```go
type URLState[T any] struct {
    key          string
    defaultValue T
    value        *Signal[T]
    debounce     time.Duration
    serialize    func(T) string
    deserialize  func(string) T
    timer        *time.Timer
}

func UseURLState[T any](key string, defaultValue T) *URLState[T] {
    // Parse initial value from current URL
    initialValue := parseFromURL[T](key, defaultValue)

    u := &URLState[T]{
        key:          key,
        defaultValue: defaultValue,
        value:        Signal(initialValue),
        serialize:    defaultSerialize[T],
        deserialize:  defaultDeserialize[T],
    }

    // Listen for popstate (browser back/forward)
    Effect(func() Cleanup {
        handler := func() {
            newValue := parseFromURL[T](key, defaultValue)
            u.value.Set(newValue)
        }

        addPopStateListener(handler)

        return func() {
            removePopStateListener(handler)
        }
    })

    return u
}

func (u *URLState[T]) Set(value T) {
    u.value.Set(value)

    if u.debounce > 0 {
        if u.timer != nil {
            u.timer.Stop()
        }
        u.timer = time.AfterFunc(u.debounce, func() {
            u.pushToURL(value)
        })
    } else {
        u.pushToURL(value)
    }
}

func (u *URLState[T]) pushToURL(value T) {
    serialized := u.serialize(value)

    // Build new URL
    url := getCurrentURL()
    if serialized == u.serialize(u.defaultValue) {
        url.Query().Del(u.key)
    } else {
        url.Query().Set(u.key, serialized)
    }

    // Push to history
    pushState(url)
}

// Type-specific serialization
func defaultSerialize[T any](v T) string {
    switch val := any(v).(type) {
    case string:
        return val
    case int:
        return strconv.Itoa(val)
    case bool:
        return strconv.FormatBool(val)
    case []string:
        return strings.Join(val, ",")
    default:
        // JSON fallback for complex types
        b, _ := json.Marshal(val)
        return string(b)
    }
}

func defaultDeserialize[T any](s string) T {
    var zero T
    switch any(zero).(type) {
    case string:
        return any(s).(T)
    case int:
        n, _ := strconv.Atoi(s)
        return any(n).(T)
    case bool:
        b, _ := strconv.ParseBool(s)
        return any(b).(T)
    case []string:
        return any(strings.Split(s, ",")).(T)
    default:
        var result T
        json.Unmarshal([]byte(s), &result)
        return result
    }
}
```

---

## 8.5 Client Hooks

### Goals

1. Enable 60fps interactions (drag-drop, sortable, tooltips)
2. Client handles animation, server owns state
3. Single event to server when interaction completes
4. Bundled standard hooks + custom hook support
5. Graceful degradation

### Core Concept

Client Hooks delegate interaction physics to JavaScript while keeping state on the server:

```
User drags card → Hook handles animation at 60fps (no network) →
User drops card → ONE event to server → Server updates DB
```

### Public API

#### Hook Attribute

```go
// Attach a hook to an element
func Hook(name string, config map[string]any) vango.Attribute

// Handle events from hooks
func OnEvent(name string, handler func(HookEvent)) vango.Attribute

// HookEvent type
type HookEvent struct {
    Name string
    Data map[string]any
}

// Type-safe accessors
func (e HookEvent) String(key string) string
func (e HookEvent) Int(key string) int
func (e HookEvent) Float(key string) float64
func (e HookEvent) Bool(key string) bool
func (e HookEvent) Strings(key string) []string
func (e HookEvent) Raw(key string) any

// Revert the client-side change (on server error)
func (e HookEvent) Revert()
```

### Standard Hooks

> **VangoUI Amendment**: Expanded to include FocusTrap and Portal hooks for Modal/Dialog/Popover support.

| Hook | Purpose | Events | Added For |
|------|---------|--------|-----------|
| `Sortable` | Drag-to-reorder lists | `reorder` | Lists/Grids |
| `Draggable` | Free-form dragging | `dragend` | Kanban/Canvas |
| `Droppable` | Drop zones | `drop` | File Upload |
| `Resizable` | Resize handles | `resize` | Panels |
| `Tooltip` | Hover tooltips | (visual only) | UI Hints |
| `Dropdown` | Click-outside-to-close | `close` | Menus |
| `Collapsible` | Expand/collapse | `toggle` | Accordions |
| `FocusTrap` | Constrain keyboard focus | (none) | Modals/Dialogs |
| `Portal` | Render at body root | (none) | Overlays |

#### Implementation Layer Notes

The `FocusTrap` and `Portal` hooks are **implementation details** for VangoUI components—not intended for direct use by end-users in most cases:

```
VangoUI Component: ui.Dialog (Go)
    │
    ├── Configuration: Uses dialogHookConfig struct to serialize params
    │
    └── Runtime: Thin Client executes FocusTrap and Portal hooks
```

This separation allows Phase 8 to focus on **capabilities** while VangoUI focuses on **developer experience**.

### Usage Examples

#### Sortable List

```go
func TaskList(tasks []Task) *vango.VNode {
    return Ul(
        Class("task-list"),

        Hook("Sortable", map[string]any{
            "animation":  150,
            "ghostClass": "sortable-ghost",
            "handle":     ".drag-handle",
        }),

        OnEvent("reorder", func(e vango.HookEvent) {
            fromIdx := e.Int("fromIndex")
            toIdx := e.Int("toIndex")

            err := db.Tasks.Reorder(fromIdx, toIdx)
            if err != nil {
                e.Revert()
                toast.Error("Failed to reorder")
            }
        }),

        Range(tasks, func(task Task, i int) *vango.VNode {
            return Li(
                Key(task.ID),
                Data("id", task.ID),

                Span(Class("drag-handle"), Text("⋮⋮")),
                Span(Text(task.Title)),
            )
        }),
    )
}
```

#### Kanban Board

```go
func KanbanBoard(columns []Column) vango.Component {
    return vango.Func(func() *vango.VNode {
        return Div(Class("kanban-board"),
            Range(columns, func(col Column, i int) *vango.VNode {
                return Div(
                    Key(col.ID),
                    Class("kanban-column"),
                    Data("column-id", col.ID),

                    Hook("Sortable", map[string]any{
                        "group":      "cards",
                        "animation":  150,
                        "ghostClass": "card-ghost",
                    }),

                    OnEvent("reorder", func(e vango.HookEvent) {
                        cardID := e.String("id")
                        toColumn := e.String("toColumn")
                        toIndex := e.Int("newIndex")

                        err := db.Cards.Move(cardID, toColumn, toIndex)
                        if err != nil {
                            e.Revert()
                            toast.Error("Failed to move card")
                        }
                    }),

                    H3(Text(col.Name)),
                    Div(Class("card-list"),
                        Range(col.Cards, CardComponent),
                    ),
                )
            }),
        )
    })
}
```

#### Tooltip

```go
Button(
    Hook("Tooltip", map[string]any{
        "content":   "Click to save your changes",
        "placement": "top",
        "delay":     200,
    }),
    OnClick(handleSave),
    Text("Save"),
)
```

#### Dropdown

```go
func DropdownMenu() vango.Component {
    return vango.Func(func() *vango.VNode {
        open := Signal(false)

        return Div(Class("dropdown"),
            Button(OnClick(open.Toggle), Text("Menu")),

            If(open(),
                Div(
                    Class("dropdown-content"),

                    Hook("Dropdown", map[string]any{
                        "closeOnEscape": true,
                        "closeOnClick":  true,
                    }),

                    OnEvent("close", func(e vango.HookEvent) {
                        open.Set(false)
                    }),

                    MenuItem("Edit", handleEdit),
                    MenuItem("Delete", handleDelete),
                ),
            ),
        )
    })
}
```

### Custom Hooks

```javascript
// public/js/hooks.js
export default {
    ColorPicker: {
        mounted(el, config, pushEvent) {
            this.picker = new Pickr({
                el: el,
                default: config.color || '#000000',
            });

            this.picker.on('change', (color) => {
                pushEvent('color-changed', {
                    color: color.toHEXA().toString()
                });
            });
        },

        updated(el, config, pushEvent) {
            if (config.color) {
                this.picker.setColor(config.color);
            }
        },

        destroyed(el) {
            this.picker.destroy();
        }
    }
};
```

```go
// Register in vango.json
{
    "hooks": "./public/js/hooks.js"
}

// Use in component
Div(
    Hook("ColorPicker", map[string]any{
        "color": currentColor,
    }),

    OnEvent("color-changed", func(e vango.HookEvent) {
        newColor := e.String("color")
        db.Settings.SetColor(newColor)
    }),
)
```

### Hook Protocol

Hooks communicate via special event types in the binary protocol:

```go
// Event type for hook events
const EventHook = 0x10

// Wire format:
// [EventHook (1 byte)]
// [HID (varint)]
// [Event name length (varint)]
// [Event name (utf8)]
// [JSON payload length (varint)]
// [JSON payload]
```

### Thin Client Hook Integration

```javascript
class VangoClient {
    hooks = {};
    hookInstances = {};

    registerHook(name, implementation) {
        this.hooks[name] = implementation;
    }

    mountHook(el, hookName, config) {
        const Hook = this.hooks[hookName];
        if (!Hook) {
            console.warn(`Unknown hook: ${hookName}`);
            return;
        }

        const hid = el.dataset.hid;
        const pushEvent = (event, data) => {
            this.sendHookEvent(hid, event, data);
        };

        const instance = Object.create(Hook);
        instance.mounted(el, config, pushEvent);
        this.hookInstances[hid] = { instance, hookName, el };
    }

    sendHookEvent(hid, eventName, data) {
        const buffer = encodeHookEvent(hid, eventName, data);
        this.ws.send(buffer);
    }
}
```

---

## 8.6 Shared State

### Goals

1. Session-scoped state (shared within one user's session)
2. Global state (shared across ALL users)
3. Same reactive model as local signals
4. Automatic synchronization

### Public API

```go
// Session-scoped signal (shared across components in one session)
func SharedSignal[T any](initial T) *Signal[T]

// Session-scoped memo
func SharedMemo[T any](compute func() T) *Memo[T]

// Global signal (shared across ALL sessions)
func GlobalSignal[T any](initial T) *Signal[T]

// Global memo
func GlobalMemo[T any](compute func() T) *Memo[T]
```

### Usage Examples

#### Session State (Shopping Cart)

```go
// store/cart.go
package store

// Shared across all components in this user's session
var CartItems = vango.SharedSignal([]CartItem{})

var CartTotal = vango.SharedMemo(func() float64 {
    total := 0.0
    for _, item := range CartItems() {
        total += item.Price * float64(item.Qty)
    }
    return total
})

var CartCount = vango.SharedMemo(func() int {
    count := 0
    for _, item := range CartItems() {
        count += item.Qty
    }
    return count
})

func AddItem(product Product, qty int) {
    CartItems.Update(func(items []CartItem) []CartItem {
        // Check if already in cart
        for i, item := range items {
            if item.ProductID == product.ID {
                items[i].Qty += qty
                return items
            }
        }
        return append(items, CartItem{
            ProductID: product.ID,
            Product:   product,
            Qty:       qty,
        })
    })
}
```

```go
// components/header.go
func Header() *vango.VNode {
    return Nav(
        Logo(),
        NavLinks(),
        // Auto-updates when cart changes
        CartBadge(store.CartCount()),
    )
}

// components/product_card.go
func ProductCard(p Product) *vango.VNode {
    return Div(
        Img(Src(p.ImageURL)),
        H3(Text(p.Name)),
        Button(
            OnClick(func() { store.AddItem(p, 1) }),
            Text("Add to Cart"),
        ),
    )
}
```

#### Global State (Real-time Presence)

```go
// store/presence.go
package store

// Shared across ALL connected users
var OnlineUsers = vango.GlobalSignal([]User{})
var CursorPositions = vango.GlobalSignal(map[string]Position{})

func UserJoined(user User) {
    OnlineUsers.Update(func(users []User) []User {
        return append(users, user)
    })
}

func UserLeft(userID string) {
    OnlineUsers.Update(func(users []User) []User {
        result := make([]User, 0, len(users))
        for _, u := range users {
            if u.ID != userID {
                result = append(result, u)
            }
        }
        return result
    })

    CursorPositions.Update(func(pos map[string]Position) map[string]Position {
        delete(pos, userID)
        return pos
    })
}

func MoveCursor(userID string, pos Position) {
    CursorPositions.Update(func(positions map[string]Position) map[string]Position {
        positions[userID] = pos
        return positions
    })
}
```

```go
// components/collaborative_canvas.go
func CollaborativeCanvas() vango.Component {
    return vango.Func(func() *vango.VNode {
        cursors := store.CursorPositions()
        currentUser := ctx.User()

        return Div(Class("canvas"),
            // Render other users' cursors
            Range(cursors, func(userID string, pos Position) *vango.VNode {
                if userID == currentUser.ID {
                    return nil // Don't show own cursor
                }
                return CursorIndicator(userID, pos)
            }),

            // Track mouse movement
            Div(
                Class("canvas-area"),
                OnMouseMove(vango.Throttle(50*time.Millisecond, func(e MouseEvent) {
                    store.MoveCursor(currentUser.ID, Position{e.ClientX, e.ClientY})
                })),
            ),
        )
    })
}
```

### Internal Implementation

```go
// SharedSignal - scoped to session
type sharedSignalRegistry struct {
    mu      sync.RWMutex
    signals map[uint64]map[string]*signalBase // session -> key -> signal
}

var sharedRegistry = &sharedSignalRegistry{
    signals: make(map[uint64]map[string]*signalBase),
}

func SharedSignal[T any](initial T) *Signal[T] {
    session := getCurrentSession()
    key := getCallerLocation() // Use source location as unique key

    sharedRegistry.mu.Lock()
    defer sharedRegistry.mu.Unlock()

    if sharedRegistry.signals[session.ID] == nil {
        sharedRegistry.signals[session.ID] = make(map[string]*signalBase)
    }

    if existing := sharedRegistry.signals[session.ID][key]; existing != nil {
        return existing.(*Signal[T])
    }

    sig := Signal(initial)
    sharedRegistry.signals[session.ID][key] = &sig.signalBase
    return sig
}

// GlobalSignal - shared across all sessions
type globalSignalRegistry struct {
    mu      sync.RWMutex
    signals map[string]*signalBase // key -> signal
}

var globalRegistry = &globalSignalRegistry{
    signals: make(map[string]*signalBase),
}

func GlobalSignal[T any](initial T) *Signal[T] {
    key := getCallerLocation()

    globalRegistry.mu.Lock()
    defer globalRegistry.mu.Unlock()

    if existing := globalRegistry.signals[key]; existing != nil {
        return existing.(*Signal[T])
    }

    sig := Signal(initial)
    globalRegistry.signals[key] = &sig.signalBase

    // Global signals notify across all sessions
    sig.onUpdate = func() {
        broadcastToAllSessions(key)
    }

    return sig
}

func broadcastToAllSessions(signalKey string) {
    sessionManager.mu.RLock()
    sessions := make([]*Session, 0, len(sessionManager.sessions))
    for _, s := range sessionManager.sessions {
        sessions = append(sessions, s)
    }
    sessionManager.mu.RUnlock()

    for _, session := range sessions {
        session.markDirtyFromGlobal(signalKey)
    }
}
```

### Persistence

```go
// Persist to browser storage
var UserPrefs = vango.SharedSignal(Preferences{}).
    Persist(vango.LocalStorage, "user-prefs")

// Persist to database
var UserSettings = vango.SharedSignal(Settings{}).
    Persist(vango.Database, "settings:" + userID)

// Custom persistence backend
var LargeData = vango.SharedSignal(Data{}).
    Persist(vango.Custom(redisStore), "large-data-key")
```

---

## 8.7 Optimistic Updates

### Goals

1. Instant visual feedback for interactions
2. Server confirms or reverts
3. Simple API for common cases
4. Works with existing signals

### Public API

```go
// Optimistic class toggle
func OptimisticClass(class string, addNotRemove bool) vango.Attribute

// Optimistic text change
func OptimisticText(newText string) vango.Attribute

// Optimistic attribute change
func OptimisticAttr(name, value string) vango.Attribute

// Optimistic on parent element
func OptimisticParentClass(class string, addNotRemove bool) vango.Attribute
```

### Usage Examples

```go
// Like button with instant feedback
func LikeButton(postID string, likes int, userLiked bool) *vango.VNode {
    return Button(
        Class("like-button"),
        ClassIf(userLiked, "liked"),

        OnClick(func() {
            if userLiked {
                db.Posts.Unlike(postID)
            } else {
                db.Posts.Like(postID)
            }
        }),

        // Instant visual feedback
        OptimisticClass("liked", !userLiked),
        OptimisticText(func() string {
            if userLiked {
                return fmt.Sprintf("%d", likes-1)
            }
            return fmt.Sprintf("%d", likes+1)
        }()),

        Icon("heart"),
        Span(Textf("%d", likes)),
    )
}

// Task checkbox
func TaskItem(task Task) *vango.VNode {
    return Li(
        Key(task.ID),
        Class("task-item"),
        ClassIf(task.Done, "completed"),

        Input(
            Type("checkbox"),
            Checked(task.Done),
            OnChange(func() {
                db.Tasks.Toggle(task.ID)
            }),
            OptimisticParentClass("completed", !task.Done),
        ),

        Span(Text(task.Title)),
    )
}
```

### How It Works

```
1. User clicks button
2. Client applies optimistic change immediately (class, text, etc.)
3. Client sends event to server
4. Server processes and sends confirmation patch
5a. If success: Server patch matches optimistic state (no visual change)
5b. If failure: Server patch reverts to original state
```

### Wire Format

Optimistic updates are encoded as attributes on the element:

```html
<button data-hid="h5"
        data-optimistic-class="liked:add"
        data-optimistic-text="43">
    <span>42</span>
</button>
```

The thin client reads these and applies them immediately on click.

---

## 8.8 JavaScript Islands

### Goals

1. Integrate third-party JavaScript libraries
2. Lazy load JS only when needed
3. Two-way communication with Go components
4. SSR placeholder rendering

### Public API

```go
// Create a JavaScript island
func JSIsland(id string, module JSModule, props JSProps) *vango.VNode

// Module reference
type JSModule string

// Props passed to JavaScript
type JSProps map[string]any

// Send message to island
func SendToIsland(id string, message map[string]any)

// Receive messages from island
func OnIslandMessage(id string, handler func(map[string]any))
```

### Usage Examples

```go
// Chart library integration
func AnalyticsDashboard(data []DataPoint) *vango.VNode {
    return Div(Class("dashboard"),
        H1(Text("Analytics")),

        JSIsland("revenue-chart",
            JSModule("/js/charts.js"),
            JSProps{
                "data":   data,
                "type":   "line",
                "height": 400,
            },
        ),
    )
}

// Rich text editor
func ArticleEditor(content string) vango.Component {
    return vango.Func(func() *vango.VNode {
        savedContent := Signal(content)

        OnIslandMessage("editor", func(msg map[string]any) {
            if msg["event"] == "change" {
                savedContent.Set(msg["content"].(string))
            }
        })

        save := func() {
            db.Articles.Save(savedContent())
        }

        return Div(
            JSIsland("editor",
                JSModule("/js/editor.js"),
                JSProps{
                    "content":     content,
                    "placeholder": "Write your article...",
                },
            ),

            Button(OnClick(save), Text("Save")),
        )
    })
}
```

### JavaScript Side

```javascript
// public/js/charts.js
import { Chart } from 'chart.js';

export function mount(container, props) {
    const chart = new Chart(container, {
        type: props.type,
        data: formatData(props.data),
        options: { maintainAspectRatio: false }
    });

    // Return cleanup function
    return () => chart.destroy();
}

export function update(container, props, state) {
    state.chart.data = formatData(props.data);
    state.chart.update();
}
```

```javascript
// public/js/editor.js
import { Editor } from '@tiptap/core';
import { sendToVango } from '@vango/bridge';

export function mount(container, props, islandId) {
    const editor = new Editor({
        element: container,
        content: props.content,
        onUpdate: ({ editor }) => {
            sendToVango(islandId, {
                event: 'change',
                content: editor.getHTML()
            });
        }
    });

    return () => editor.destroy();
}
```

### SSR Behavior

Islands render as placeholders during SSR:

```html
<div id="revenue-chart"
     data-island="true"
     data-island-module="/js/charts.js"
     data-island-props='{"type":"line","height":400}'>
    <div class="island-loading">Loading chart...</div>
</div>
```

The thin client hydrates after connecting:

```javascript
document.querySelectorAll('[data-island]').forEach(async (el) => {
    const mod = await import(el.dataset.islandModule);
    const props = JSON.parse(el.dataset.islandProps);
    el._cleanup = mod.mount(el, props, el.id);
});
```

---

## Testing Strategy

### Unit Tests

Each subsystem has isolated tests:

```go
// Form tests
func TestFormValidation(t *testing.T) {
    form := UseForm(ContactForm{})

    // Empty form should fail validation
    assert.False(t, form.Validate())
    assert.True(t, form.HasError("Name"))
    assert.True(t, form.HasError("Email"))

    // Fill required fields
    form.Set("Name", "John")
    form.Set("Email", "john@example.com")

    assert.True(t, form.Validate())
    assert.False(t, form.HasError("Name"))
}

// Resource tests
func TestResourceStates(t *testing.T) {
    fetched := false
    resource := Resource(func() (string, error) {
        fetched = true
        return "data", nil
    })

    // Initially loading
    assert.True(t, resource.IsLoading())

    // Wait for fetch
    time.Sleep(10 * time.Millisecond)

    assert.True(t, fetched)
    assert.True(t, resource.IsReady())
    assert.Equal(t, "data", resource.Data())
}

// Context tests
func TestContextPropagation(t *testing.T) {
    ctx := CreateContext("default")

    // Without provider, get default
    value := ctx.Use()
    assert.Equal(t, "default", value)

    // With provider, get provided value
    withProvider(ctx, "custom", func() {
        value := ctx.Use()
        assert.Equal(t, "custom", value)
    })
}
```

### Integration Tests

```go
func TestFormSubmission(t *testing.T) {
    app := vango.TestApp()

    page := app.Navigate("/contact")

    // Fill form
    page.Fill("[name=name]", "John Doe")
    page.Fill("[name=email]", "john@example.com")
    page.Fill("[name=message]", "Hello!")

    // Submit
    page.Click("[type=submit]")

    // Assert success
    assert.Contains(t, page.Text(), "Message sent")
}

func TestResourceLoading(t *testing.T) {
    app := vango.TestApp()

    page := app.Navigate("/users/123")

    // Initially shows loading
    assert.Contains(t, page.Text(), "Loading")

    // Wait for data
    page.WaitFor("[data-testid=user-card]")

    // Shows user data
    assert.Contains(t, page.Text(), "John Doe")
}
```

---

## File Structure

```
pkg/features/
├── form/
│   ├── form.go
│   ├── form_test.go
│   ├── validators.go
│   ├── validators_test.go
│   ├── array.go
│   └── array_test.go
├── resource/
│   ├── resource.go
│   ├── resource_test.go
│   ├── match.go
│   └── options.go
├── context/
│   ├── context.go
│   └── context_test.go
├── urlstate/
│   ├── urlstate.go
│   ├── urlstate_test.go
│   ├── hashstate.go
│   └── serialize.go
├── hooks/
│   ├── hook.go
│   ├── hook_test.go
│   ├── event.go
│   └── standard/
│       ├── sortable.go
│       ├── draggable.go
│       ├── tooltip.go
│       └── dropdown.go
├── shared/
│   ├── shared_signal.go
│   ├── global_signal.go
│   ├── shared_test.go
│   └── persistence.go
├── optimistic/
│   ├── optimistic.go
│   └── optimistic_test.go
└── islands/
    ├── island.go
    ├── island_test.go
    └── bridge.go
```

---

## Exit Criteria

Phase 8 is complete when:

1. [x] Forms: UseForm with validation, arrays, all built-in validators
2. [x] Resources: Resource with all states, Match helper, caching
3. [x] Context: CreateContext, Provider, Use working correctly
4. [x] URL State: UseURLState, UseHashState, serialization
5. [x] Client Hooks: Hook attribute, OnEvent, standard hooks bundled
6. [x] Shared State: SharedSignal, GlobalSignal, broadcast working
7. [x] Optimistic: OptimisticClass, OptimisticText working
8. [x] JS Islands: JSIsland, two-way communication, SSR placeholders
9. [x] All subsystems have unit tests with > 80% coverage
10. [x] Integration tests for common workflows
11. [x] Documentation for each subsystem

**All exit criteria verified complete on 2024-12-07.**

---

## Test Coverage Summary

| Package | Coverage | Location |
|---------|----------|----------|
| context | 100.0% | `pkg/features/context/` |
| form | 80.1% | `pkg/features/form/` |
| hooks | 100.0% | `pkg/features/hooks/` |
| hooks/standard | 100.0% | `pkg/features/hooks/standard/` |
| islands | 100.0% | `pkg/features/islands/` |
| optimistic | 100.0% | `pkg/features/optimistic/` |
| resource | 85.8% | `pkg/features/resource/` |
| store | 96.4% | `pkg/features/store/` |
| urlstate | 92.7% | `pkg/features/urlstate/` |

Integration tests: `pkg/features/integration_test.go` (14 tests)

---

## Dependencies

- **Requires**: Phases 1-7 (all foundation and integration layers)
- **Required by**: Phase 9 (Developer Experience), Phase 10 (Production)

---

*Phase 8 Specification - Version 1.0 - Completed 2024-12-07*
