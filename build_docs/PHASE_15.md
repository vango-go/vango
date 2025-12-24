# Phase 15: VangoUI Component System

> **A CLI-distributed, server-first component library with intelligent client hooks**

---

## Overview

Phase 15 introduces VangoUI, a component library designed specifically for Vango's server-driven architecture. Unlike traditional component libraries that are installed as dependencies, VangoUI components are copied into your project—you own the source code completely.

### Core Philosophy

**Components are code you own, not dependencies you import.**

### Goals

1. **CLI Distribution**: Components copied via `vango add`, not imported
2. **Server-First**: Most components require zero JavaScript
3. **Type Safety**: Functional options pattern for compile-time validation
4. **Client Hooks**: Standardized protocol for interactive components
5. **Theme System**: CSS variables for consistent styling

### Non-Goals (Explicit Exclusions)

1. Runtime component library (components are source code)
2. React/Vue compatibility layer (Vango-native only)
3. Design tool plugins (Figma, Sketch)

### Design Principles

| Principle | Implementation |
|-----------|----------------|
| Code Ownership | You can edit any component |
| Type Safety | Invalid options don't compile |
| Server-First | Primitives have 0KB client JS |
| Zero Bloat | Only what you use |
| AI-Optimized | Single-file components |

---

## Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 15.1 CLI Integration | `vango add init` and `vango add <component>` | Critical |
| 15.2 Component Registry | Manifest and versioning | Critical |
| 15.3 Styling System | Tailwind, CN utility, CSS variables | Critical |
| 15.4 Functional Options API | Type-safe component configuration | Critical |
| 15.5 Client Hook Protocol | Interactive component communication | High |
| 15.6 Standard Hooks | Dialog, Popover, Sortable, etc. | High |
| 15.7 Component Library | Full component set | High |

---

## 15.1 CLI Integration

### Command: `vango add init`

Initializes VangoUI in an existing Vango project:

```bash
vango add init
```

#### Actions Performed

1. Creates `app/components/ui/utils.go`
2. Creates/updates `tailwind.config.js`
3. Updates `public/styles.css` with CSS variables
4. Creates `.vscode/settings.json` for Tailwind IntelliSense

#### Generated `utils.go`

```go
// app/components/ui/utils.go

package ui

import (
    "strings"
    
    "github.com/vango-dev/vango"
    "github.com/vango-dev/tailwind-merge-go"
)

// Type aliases for convenience
type VNode = vango.VNode
type Signal[T any] = vango.Signal[T]
type Ctx = vango.Ctx

// CN merges Tailwind classes intelligently, handling conflicts.
// Example: CN("p-4 p-2") -> "p-2" (last wins)
// Example: CN("text-red-500", "text-blue-500") -> "text-blue-500"
func CN(classes ...string) string {
    return tailwind.Merge(strings.Join(classes, " "))
}

// Shared option types (used across components)
type Size string

const (
    SizeSm   Size = "sm"
    SizeMd   Size = "md"
    SizeLg   Size = "lg"
    SizeIcon Size = "icon"
)

type Variant string

const (
    VariantDefault     Variant = "default"
    VariantDestructive Variant = "destructive"
    VariantOutline     Variant = "outline"
    VariantSecondary   Variant = "secondary"
    VariantGhost       Variant = "ghost"
    VariantLink        Variant = "link"
)
```

#### Generated Tailwind Config

```javascript
// tailwind.config.js

/** @type {import('tailwindcss').Config} */
module.exports = {
    darkMode: ["class"],
    content: [
        "./app/**/*.go",
        "./public/**/*.html",
    ],
    theme: {
        container: {
            center: true,
            padding: "2rem",
            screens: {
                "2xl": "1400px",
            },
        },
        extend: {
            colors: {
                border: "hsl(var(--border))",
                input: "hsl(var(--input))",
                ring: "hsl(var(--ring))",
                background: "hsl(var(--background))",
                foreground: "hsl(var(--foreground))",
                primary: {
                    DEFAULT: "hsl(var(--primary))",
                    foreground: "hsl(var(--primary-foreground))",
                },
                secondary: {
                    DEFAULT: "hsl(var(--secondary))",
                    foreground: "hsl(var(--secondary-foreground))",
                },
                destructive: {
                    DEFAULT: "hsl(var(--destructive))",
                    foreground: "hsl(var(--destructive-foreground))",
                },
                muted: {
                    DEFAULT: "hsl(var(--muted))",
                    foreground: "hsl(var(--muted-foreground))",
                },
                accent: {
                    DEFAULT: "hsl(var(--accent))",
                    foreground: "hsl(var(--accent-foreground))",
                },
                popover: {
                    DEFAULT: "hsl(var(--popover))",
                    foreground: "hsl(var(--popover-foreground))",
                },
                card: {
                    DEFAULT: "hsl(var(--card))",
                    foreground: "hsl(var(--card-foreground))",
                },
            },
            borderRadius: {
                lg: "var(--radius)",
                md: "calc(var(--radius) - 2px)",
                sm: "calc(var(--radius) - 4px)",
            },
            keyframes: {
                "accordion-down": {
                    from: { height: 0 },
                    to: { height: "var(--radix-accordion-content-height)" },
                },
                "accordion-up": {
                    from: { height: "var(--radix-accordion-content-height)" },
                    to: { height: 0 },
                },
            },
            animation: {
                "accordion-down": "accordion-down 0.2s ease-out",
                "accordion-up": "accordion-up 0.2s ease-out",
            },
        },
    },
    plugins: [require("tailwindcss-animate")],
}
```

#### Generated VS Code Settings

```json
// .vscode/settings.json

{
    "tailwindCSS.experimental.classRegex": [
        ["Class\\(\"([^\"]*)\"\\)", "\"([^\"]*)\""],
        ["CN\\(([^)]*)\\)", "\"([^\"]*)\""],
        ["className:\\s*\"([^\"]*)\"", "\"([^\"]*)\""]
    ],
    "tailwindCSS.includeLanguages": {
        "go": "html"
    },
    "editor.quickSuggestions": {
        "strings": true
    }
}
```

### Command: `vango add <component>`

Adds one or more components to your project:

```bash
# Single component
vango add button

# Multiple components
vango add button card input label

# View changes before applying
vango add dialog --diff

# Force overwrite existing
vango add button --force

# Check for updates
vango add --check
```

#### Add Process

```go
// cmd/vango/commands/add.go

func addComponent(name string, opts AddOptions) error {
    // 1. Fetch component manifest
    manifest, err := fetchManifest()
    if err != nil {
        return err
    }
    
    component, ok := manifest.Components[name]
    if !ok {
        return fmt.Errorf("unknown component: %s", name)
    }
    
    // 2. Check dependencies
    for _, dep := range component.Dependencies {
        depPath := filepath.Join("app/components/ui", dep+".go")
        if !fileExists(depPath) {
            fmt.Printf("Component %s requires %s. Add it? [Y/n] ", name, dep)
            if confirm() {
                if err := addComponent(dep, opts); err != nil {
                    return err
                }
            }
        }
    }
    
    // 3. Check if component exists
    destPath := filepath.Join("app/components/ui", name+".go")
    if fileExists(destPath) && !opts.Force {
        if opts.Diff {
            showDiff(destPath, component.Source)
            return nil
        }
        return fmt.Errorf("component %s already exists. Use --force to overwrite", name)
    }
    
    // 4. Fetch and write component source
    source, err := fetchComponentSource(name)
    if err != nil {
        return err
    }
    
    if opts.Diff {
        showDiff(destPath, source)
        return nil
    }
    
    // Backup existing if overwriting
    if fileExists(destPath) {
        backupPath := destPath + ".bak"
        os.Rename(destPath, backupPath)
        fmt.Printf("Backed up existing to %s\n", backupPath)
    }
    
    if err := os.WriteFile(destPath, source, 0644); err != nil {
        return err
    }
    
    // 5. Ensure hooks are available
    for _, hook := range component.Hooks {
        ensureHook(hook)
    }
    
    fmt.Printf("✅ Added %s to app/components/ui/%s.go\n", name, name)
    return nil
}
```

---

## 15.2 Component Registry

### Registry Manifest

The CLI fetches from a central manifest (or embedded in CLI):

```json
{
    "version": "1.0.0",
    "components": {
        "button": {
            "version": "1.0.0",
            "files": ["button.go"],
            "dependencies": [],
            "hooks": [],
            "description": "Primary button component with variants"
        },
        "input": {
            "version": "1.0.0",
            "files": ["input.go"],
            "dependencies": [],
            "hooks": [],
            "description": "Text input with label and error states"
        },
        "dialog": {
            "version": "1.0.0",
            "files": ["dialog.go"],
            "dependencies": ["button"],
            "hooks": ["Dialog"],
            "description": "Modal dialog with trigger and content",
            "changelog": "Initial release"
        },
        "dropdown": {
            "version": "1.0.0",
            "files": ["dropdown.go"],
            "dependencies": ["button"],
            "hooks": ["Popover"],
            "description": "Dropdown menu with keyboard navigation"
        },
        "combobox": {
            "version": "1.0.0",
            "files": ["combobox.go"],
            "dependencies": ["input", "popover"],
            "hooks": ["Combobox", "Popover"],
            "description": "Searchable select with async options"
        },
        "data-table": {
            "version": "1.0.0",
            "files": ["data-table.go"],
            "dependencies": ["button", "input", "dropdown"],
            "hooks": [],
            "description": "Server-powered data table with sorting and filtering"
        }
    },
    "hooks": {
        "Dialog": {
            "file": "dialog.js",
            "size": 1200
        },
        "Popover": {
            "file": "popover.js",
            "size": 1800
        },
        "Combobox": {
            "file": "combobox.js",
            "size": 2400
        },
        "Sortable": {
            "file": "sortable.js",
            "size": 3200
        }
    }
}
```

### Version Checking

```bash
vango add --check

# Output:
# Component Status:
# ✅ button     1.0.0 (up to date)
# ⚠️  dialog    1.0.0 → 1.1.0 (minor update available)
# ❌ dropdown   1.0.0 → 2.0.0 (breaking changes)
#
# Run 'vango add <component> --diff' to see changes.
```

---

## 15.3 Styling System

### CSS Variables

All components use CSS variables for theming:

```css
/* public/styles.css - CSS Variables Section */

:root {
    /* Light mode (default) */
    --background: 0 0% 100%;
    --foreground: 0 0% 3.9%;
    
    --card: 0 0% 100%;
    --card-foreground: 0 0% 3.9%;
    
    --popover: 0 0% 100%;
    --popover-foreground: 0 0% 3.9%;
    
    --primary: 0 0% 9%;
    --primary-foreground: 0 0% 98%;
    
    --secondary: 0 0% 96.1%;
    --secondary-foreground: 0 0% 9%;
    
    --muted: 0 0% 96.1%;
    --muted-foreground: 0 0% 45.1%;
    
    --accent: 0 0% 96.1%;
    --accent-foreground: 0 0% 9%;
    
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 0 0% 98%;
    
    --border: 0 0% 89.8%;
    --input: 0 0% 89.8%;
    --ring: 0 0% 3.9%;
    
    --radius: 0.5rem;
}

.dark {
    --background: 0 0% 3.9%;
    --foreground: 0 0% 98%;
    
    --card: 0 0% 3.9%;
    --card-foreground: 0 0% 98%;
    
    --popover: 0 0% 3.9%;
    --popover-foreground: 0 0% 98%;
    
    --primary: 0 0% 98%;
    --primary-foreground: 0 0% 9%;
    
    --secondary: 0 0% 14.9%;
    --secondary-foreground: 0 0% 98%;
    
    --muted: 0 0% 14.9%;
    --muted-foreground: 0 0% 63.9%;
    
    --accent: 0 0% 14.9%;
    --accent-foreground: 0 0% 98%;
    
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 0 0% 98%;
    
    --border: 0 0% 14.9%;
    --input: 0 0% 14.9%;
    --ring: 0 0% 83.1%;
}
```

### CN Utility

The `CN` function intelligently merges Tailwind classes:

```go
// Conflict resolution - last wins
CN("p-4", "p-2")                    // → "p-2"
CN("text-red-500", "text-blue-500") // → "text-blue-500"

// Safe concatenation - no conflicts
CN("p-4", "m-2", "rounded")         // → "p-4 m-2 rounded"

// Common pattern - base + overrides
CN(baseClasses, props.ClassName)    // → merged result
```

---

## 15.4 Functional Options API

### Pattern Overview

Every component uses a functional options pattern for configuration:

```go
// Component definition
func Button(opts ...ButtonOption) *vango.VNode

// Option type
type ButtonOption func(*buttonConfig)

// Internal config
type buttonConfig struct {
    variant   Variant
    size      Size
    disabled  bool
    onClick   func()
    children  []any
    className string
}
```

### Option Implementation

```go
// app/components/ui/button.go

package ui

import (
    "github.com/vango-dev/vango"
    . "github.com/vango-dev/vango/vdom"
)

// ButtonOption configures a Button component.
type ButtonOption func(*buttonConfig)

type buttonConfig struct {
    variant   Variant
    size      Size
    disabled  bool
    loading   bool
    onClick   func()
    children  []any
    className string
    asChild   bool
}

func defaultButtonConfig() buttonConfig {
    return buttonConfig{
        variant: VariantDefault,
        size:    SizeMd,
    }
}

// Variant options (using the "Shared Type, Local Method" pattern)
func (v Variant) ButtonOption() ButtonOption {
    return func(c *buttonConfig) {
        c.variant = v
    }
}

// Convenience: Direct variant options
var (
    Primary     = VariantDefault.ButtonOption()
    Destructive = VariantDestructive.ButtonOption()
    Outline     = VariantOutline.ButtonOption()
    Secondary   = VariantSecondary.ButtonOption()
    Ghost       = VariantGhost.ButtonOption()
    Link        = VariantLink.ButtonOption()
)

// Size options
func (s Size) ButtonOption() ButtonOption {
    return func(c *buttonConfig) {
        c.size = s
    }
}

var (
    Sm   = SizeSm.ButtonOption()
    Md   = SizeMd.ButtonOption()
    Lg   = SizeLg.ButtonOption()
    Icon = SizeIcon.ButtonOption()
)

// Behavior options
func Disabled(d bool) ButtonOption {
    return func(c *buttonConfig) {
        c.disabled = d
    }
}

func Loading(l bool) ButtonOption {
    return func(c *buttonConfig) {
        c.loading = l
    }
}

func OnClick(handler func()) ButtonOption {
    return func(c *buttonConfig) {
        c.onClick = handler
    }
}

func Children(children ...any) ButtonOption {
    return func(c *buttonConfig) {
        c.children = children
    }
}

func Class(className string) ButtonOption {
    return func(c *buttonConfig) {
        c.className = className
    }
}

// Button renders a button element with the configured options.
func Button(opts ...ButtonOption) *vango.VNode {
    cfg := defaultButtonConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    
    // Build class string
    baseClasses := "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
    
    variantClasses := map[Variant]string{
        VariantDefault:     "bg-primary text-primary-foreground hover:bg-primary/90",
        VariantDestructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        VariantOutline:     "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        VariantSecondary:   "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        VariantGhost:       "hover:bg-accent hover:text-accent-foreground",
        VariantLink:        "text-primary underline-offset-4 hover:underline",
    }
    
    sizeClasses := map[Size]string{
        SizeSm:   "h-9 rounded-md px-3",
        SizeMd:   "h-10 px-4 py-2",
        SizeLg:   "h-11 rounded-md px-8",
        SizeIcon: "h-10 w-10",
    }
    
    classes := CN(
        baseClasses,
        variantClasses[cfg.variant],
        sizeClasses[cfg.size],
        cfg.className,
    )
    
    // Build attributes
    attrs := []any{
        vdom.Class(classes),
    }
    
    if cfg.disabled || cfg.loading {
        attrs = append(attrs, vdom.Disabled(true))
    }
    
    if cfg.onClick != nil && !cfg.disabled && !cfg.loading {
        attrs = append(attrs, vdom.OnClick(cfg.onClick))
    }
    
    // Build children
    children := cfg.children
    if cfg.loading {
        children = append([]any{spinnerIcon()}, children...)
    }
    
    return vdom.Button(append(attrs, children...)...)
}

func spinnerIcon() *vango.VNode {
    return Svg(
        vdom.Class("mr-2 h-4 w-4 animate-spin"),
        // SVG path for spinner...
    )
}
```

### Usage Examples

```go
// Simple button
ui.Button(ui.Children(ui.Text("Click me")))

// Destructive button with handler
ui.Button(
    ui.Destructive,
    ui.OnClick(handleDelete),
    ui.Children(ui.Text("Delete")),
)

// Loading state
ui.Button(
    ui.Primary,
    ui.Loading(isSubmitting.Get()),
    ui.Disabled(isSubmitting.Get()),
    ui.OnClick(handleSubmit),
    ui.Children(ui.Text("Submit")),
)

// Icon button
ui.Button(
    ui.Ghost,
    ui.Icon,
    ui.Children(lucide.Menu),
)

// Custom classes
ui.Button(
    ui.Outline,
    ui.Class("w-full"),
    ui.Children(ui.Text("Full Width")),
)
```

---

## 15.5 Client Hook Protocol

### Overview

Client Hooks enable rich interactions that require JavaScript (drag-and-drop, focus trapping, keyboard navigation) while keeping state on the server.

### Wire Format

```
Hook Attachment:
┌──────────────────────────────────────────────────────────┐
│  Hook(name string, config any) vdom.Attr                 │
│                                                          │
│  Renders as: data-hook="name" data-hook-config="{...}"   │
└──────────────────────────────────────────────────────────┘

Client Event:
┌──────────────────────────────────────────────────────────┐
│  EventType: Hook (0x0A)                                  │
│  TargetHID: [varint]                                     │
│  HookName:  [string]                                     │
│  EventName: [string]                                     │
│  Payload:   [json]                                       │
└──────────────────────────────────────────────────────────┘
```

### JavaScript Contract

Every hook must implement this interface:

```typescript
// client/src/hooks/types.ts

interface VangoHook {
    /**
     * Called when the element is mounted to the DOM.
     * @param el The DOM element with data-hook attribute
     * @param config Parsed data-hook-config JSON
     * @param api API for communicating with server
     */
    mounted(el: HTMLElement, config: any, api: HookAPI): void;
    
    /**
     * Called when the element's config changes (server re-render).
     * @param el The DOM element
     * @param config New config value
     * @param api API for communicating with server
     */
    updated?(el: HTMLElement, config: any, api: HookAPI): void;
    
    /**
     * Called when the element is removed from the DOM.
     * @param el The DOM element being removed
     */
    destroyed?(el: HTMLElement): void;
}

interface HookAPI {
    /**
     * Send event to server and wait for patches.
     */
    pushEvent(eventName: string, payload: any): Promise<void>;
    
    /**
     * Send event but don't wait for response (fire-and-forget).
     */
    pushEventAsync(eventName: string, payload: any): void;
    
    /**
     * Get current config value (may change if server re-renders).
     */
    getConfig(): any;
}
```

### Go API

```go
// pkg/vdom/hook.go

// Hook attaches a client hook to an element.
func Hook(name string, config any) Attr {
    configJSON, _ := json.Marshal(config)
    return Attrs(
        DataAttr("hook", name),
        DataAttr("hook-config", string(configJSON)),
    )
}

// OnEvent registers a handler for hook events.
func OnEvent(hookName, eventName string, handler func(payload map[string]any)) Attr {
    return hookHandler{
        hookName:  hookName,
        eventName: eventName,
        handler:   handler,
    }
}

// HookEvent provides access to hook event details in handlers.
type HookEvent struct {
    HookName  string
    EventName string
    Payload   map[string]any
    revert    func()
}

// Revert undoes any optimistic updates made by the client.
func (e *HookEvent) Revert() {
    if e.revert != nil {
        e.revert()
    }
}
```

### Hook Loading Strategy

Hooks are loaded on-demand when first encountered:

```javascript
// client/src/hooks/loader.js

class HookLoader {
    constructor() {
        this.hooks = new Map();
        this.loading = new Map();
    }
    
    async load(hookName) {
        // Already loaded
        if (this.hooks.has(hookName)) {
            return this.hooks.get(hookName);
        }
        
        // Currently loading
        if (this.loading.has(hookName)) {
            return this.loading.get(hookName);
        }
        
        // Start loading
        const promise = import(`./hooks/${hookName}.js`)
            .then(module => {
                const hook = module.default;
                this.hooks.set(hookName, hook);
                this.loading.delete(hookName);
                return hook;
            });
        
        this.loading.set(hookName, promise);
        return promise;
    }
    
    async ensureLoaded(hookNames) {
        await Promise.all(hookNames.map(name => this.load(name)));
    }
}

// Preload hints for faster loading
// <link rel="modulepreload" href="/vango/hooks/Dialog.js">
```

---

## 15.6 Standard Hooks

### Dialog Hook

Handles focus trapping and keyboard dismissal:

```javascript
// client/src/hooks/Dialog.js

export default {
    mounted(el, config, api) {
        this.el = el;
        this.previousFocus = document.activeElement;
        
        // Find focusable elements
        this.focusables = el.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        
        // Focus first element
        if (this.focusables.length > 0) {
            this.focusables[0].focus();
        }
        
        // Trap focus
        el.addEventListener('keydown', this.handleKeyDown.bind(this));
        
        // Click outside to close
        if (config.closeOnClickOutside !== false) {
            document.addEventListener('click', this.handleClickOutside.bind(this));
        }
    },
    
    handleKeyDown(e) {
        if (e.key === 'Escape') {
            this.api.pushEvent('close', {});
            return;
        }
        
        if (e.key === 'Tab') {
            const first = this.focusables[0];
            const last = this.focusables[this.focusables.length - 1];
            
            if (e.shiftKey && document.activeElement === first) {
                e.preventDefault();
                last.focus();
            } else if (!e.shiftKey && document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        }
    },
    
    handleClickOutside(e) {
        if (!this.el.contains(e.target)) {
            this.api.pushEvent('close', {});
        }
    },
    
    destroyed() {
        document.removeEventListener('click', this.handleClickOutside);
        if (this.previousFocus) {
            this.previousFocus.focus();
        }
    }
};
```

### Popover Hook

Handles positioning and click-outside:

```javascript
// client/src/hooks/Popover.js

export default {
    mounted(el, config, api) {
        this.trigger = el.querySelector('[data-popover-trigger]');
        this.content = el.querySelector('[data-popover-content]');
        this.api = api;
        
        if (!this.trigger || !this.content) return;
        
        // Position content
        this.position();
        
        // Reposition on scroll/resize
        window.addEventListener('scroll', this.position.bind(this), true);
        window.addEventListener('resize', this.position.bind(this));
        
        // Click outside
        document.addEventListener('click', this.handleClickOutside.bind(this));
    },
    
    position() {
        const triggerRect = this.trigger.getBoundingClientRect();
        const contentRect = this.content.getBoundingClientRect();
        
        // Default: below trigger, aligned left
        let top = triggerRect.bottom + 4;
        let left = triggerRect.left;
        
        // Flip if would overflow
        if (top + contentRect.height > window.innerHeight) {
            top = triggerRect.top - contentRect.height - 4;
        }
        
        if (left + contentRect.width > window.innerWidth) {
            left = triggerRect.right - contentRect.width;
        }
        
        this.content.style.position = 'fixed';
        this.content.style.top = `${top}px`;
        this.content.style.left = `${left}px`;
    },
    
    handleClickOutside(e) {
        if (!this.el.contains(e.target)) {
            this.api.pushEvent('close', {});
        }
    },
    
    destroyed() {
        window.removeEventListener('scroll', this.position, true);
        window.removeEventListener('resize', this.position);
        document.removeEventListener('click', this.handleClickOutside);
    }
};
```

### Sortable Hook

Handles drag-and-drop reordering:

```javascript
// client/src/hooks/Sortable.js

export default {
    mounted(el, config, api) {
        this.el = el;
        this.api = api;
        this.items = el.querySelectorAll('[data-sortable-item]');
        
        this.items.forEach(item => {
            item.draggable = true;
            item.addEventListener('dragstart', this.handleDragStart.bind(this));
            item.addEventListener('dragover', this.handleDragOver.bind(this));
            item.addEventListener('drop', this.handleDrop.bind(this));
            item.addEventListener('dragend', this.handleDragEnd.bind(this));
        });
    },
    
    handleDragStart(e) {
        this.dragging = e.target.closest('[data-sortable-item]');
        this.dragging.classList.add('opacity-50');
        e.dataTransfer.effectAllowed = 'move';
    },
    
    handleDragOver(e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        
        const target = e.target.closest('[data-sortable-item]');
        if (!target || target === this.dragging) return;
        
        const rect = target.getBoundingClientRect();
        const midY = rect.top + rect.height / 2;
        
        if (e.clientY < midY) {
            target.parentNode.insertBefore(this.dragging, target);
        } else {
            target.parentNode.insertBefore(this.dragging, target.nextSibling);
        }
    },
    
    handleDrop(e) {
        e.preventDefault();
        
        // Collect new order
        const newOrder = Array.from(this.el.querySelectorAll('[data-sortable-item]'))
            .map(item => item.dataset.sortableId);
        
        // Send to server
        this.api.pushEvent('reorder', { order: newOrder });
    },
    
    handleDragEnd() {
        this.dragging.classList.remove('opacity-50');
        this.dragging = null;
    },
    
    updated(el, config) {
        // Re-bind items if server re-rendered
        this.items = el.querySelectorAll('[data-sortable-item]');
    }
};
```

---

## 15.7 Component Library

### Component Categories

| Category | Components | Client JS |
|----------|------------|-----------|
| **Primitives** | Button, Badge, Label, Separator, Skeleton | 0 KB |
| **Form** | Input, Textarea, Select, Checkbox, Radio, Switch, Slider | 0 KB |
| **Layout** | Card, Accordion, Tabs, Collapsible | ~1 KB |
| **Interactive** | Dialog, Dropdown, Popover, Sheet, Command | 2-4 KB |
| **Data** | DataTable, Avatar, Progress | 0 KB |

### Primitive Components

#### Card

```go
// app/components/ui/card.go

package ui

import (
    . "github.com/vango-dev/vango/vdom"
)

type CardOption func(*cardConfig)

type cardConfig struct {
    className string
    children  []any
}

func (c cardConfig) HeaderOpt() CardHeaderOption { return func(cfg *cardHeaderConfig) { cfg.className = c.className } }

func Card(opts ...CardOption) *VNode {
    cfg := cardConfig{}
    for _, opt := range opts {
        opt(&cfg)
    }
    
    return Div(
        Class(CN("rounded-lg border bg-card text-card-foreground shadow-sm", cfg.className)),
        cfg.children...,
    )
}

// CardHeader, CardTitle, CardDescription, CardContent, CardFooter follow same pattern...
```

#### Badge

```go
// app/components/ui/badge.go

package ui

type BadgeVariant string

const (
    BadgeDefault     BadgeVariant = "default"
    BadgeSecondary   BadgeVariant = "secondary"
    BadgeDestructive BadgeVariant = "destructive"
    BadgeOutline     BadgeVariant = "outline"
)

type BadgeOption func(*badgeConfig)

type badgeConfig struct {
    variant   BadgeVariant
    className string
    children  []any
}

func Badge(opts ...BadgeOption) *VNode {
    cfg := badgeConfig{variant: BadgeDefault}
    for _, opt := range opts {
        opt(&cfg)
    }
    
    variantClasses := map[BadgeVariant]string{
        BadgeDefault:     "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        BadgeSecondary:   "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        BadgeDestructive: "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
        BadgeOutline:     "text-foreground",
    }
    
    return Span(
        Class(CN(
            "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors",
            variantClasses[cfg.variant],
            cfg.className,
        )),
        cfg.children...,
    )
}
```

### Interactive Components

#### Dialog

```go
// app/components/ui/dialog.go

package ui

type DialogOption func(*dialogConfig)

type dialogConfig struct {
    open        *Signal[bool]
    onClose     func()
    trigger     *VNode
    content     *VNode
    title       string
    description string
}

func DialogOpen(s *Signal[bool]) DialogOption {
    return func(c *dialogConfig) { c.open = s }
}

func DialogOnClose(fn func()) DialogOption {
    return func(c *dialogConfig) { c.onClose = fn }
}

func DialogTrigger(node *VNode) DialogOption {
    return func(c *dialogConfig) { c.trigger = node }
}

func DialogContent(node *VNode) DialogOption {
    return func(c *dialogConfig) { c.content = node }
}

func DialogTitle(title string) DialogOption {
    return func(c *dialogConfig) { c.title = title }
}

func DialogDescription(desc string) DialogOption {
    return func(c *dialogConfig) { c.description = desc }
}

func Dialog(opts ...DialogOption) *VNode {
    cfg := dialogConfig{}
    for _, opt := range opts {
        opt(&cfg)
    }
    
    // Handle hook events
    handleClose := func(payload map[string]any) {
        if cfg.open != nil {
            cfg.open.Set(false)
        }
        if cfg.onClose != nil {
            cfg.onClose()
        }
    }
    
    return Div(
        // Trigger
        Div(
            OnClick(func() { cfg.open.Set(true) }),
            cfg.trigger,
        ),
        
        // Portal overlay + dialog
        If(cfg.open != nil && cfg.open.Get(),
            Div(
                Class("fixed inset-0 z-50 bg-black/80"),
                OnClick(handleClose),
            ),
            Div(
                Class("fixed left-[50%] top-[50%] z-50 translate-x-[-50%] translate-y-[-50%] grid w-full max-w-lg gap-4 border bg-background p-6 shadow-lg sm:rounded-lg"),
                Hook("Dialog", map[string]any{"closeOnClickOutside": true}),
                OnEvent("Dialog", "close", handleClose),
                
                // Title
                If(cfg.title != "",
                    H2(Class("text-lg font-semibold leading-none"), Text(cfg.title)),
                ),
                
                // Description
                If(cfg.description != "",
                    P(Class("text-sm text-muted-foreground"), Text(cfg.description)),
                ),
                
                // Content
                cfg.content,
            ),
        ),
    )
}
```

---

## Exit Criteria

### 15.1 CLI Integration
- [ ] `vango add init` creates utils.go, tailwind.config.js, CSS variables
- [ ] `vango add init` creates VS Code settings for Tailwind IntelliSense
- [ ] `vango add button` downloads and writes button.go
- [ ] `vango add --diff` shows changes without applying
- [ ] `vango add --force` overwrites with backup
- [ ] Dependency resolution prompts for missing components

### 15.2 Component Registry
- [ ] Manifest includes all components with versions
- [ ] Manifest includes dependency graph
- [ ] Manifest includes required hooks
- [ ] `vango add --check` shows available updates

### 15.3 Styling System
- [ ] CSS variables cover all theme colors
- [ ] Dark mode variables defined
- [ ] CN utility merges classes correctly
- [ ] Tailwind config extends with VangoUI tokens

### 15.4 Functional Options API
- [ ] Button component uses functional options
- [ ] Shared types (Variant, Size) work across components
- [ ] Invalid options cause compile errors
- [ ] All options are discoverable via IDE

### 15.5 Client Hook Protocol
- [ ] Hook wire format documented
- [ ] Hook loader implemented in thin client
- [ ] pushEvent sends correctly formatted messages
- [ ] HookEvent.Revert() works for optimistic updates

### 15.6 Standard Hooks
- [ ] Dialog hook: focus trap, Escape key, click outside
- [ ] Popover hook: positioning, click outside
- [ ] Sortable hook: drag-drop reorder, server sync
- [ ] All hooks have destroyed() cleanup

### 15.7 Component Library
- [ ] Primitives: Button, Badge, Label, Separator, Skeleton
- [ ] Form: Input, Textarea, Checkbox, Switch
- [ ] Layout: Card, Accordion, Tabs
- [ ] Interactive: Dialog, Dropdown, Popover
- [ ] Data: DataTable, Avatar, Progress

---

## Files Changed

### New Files
- `client/src/hooks/loader.js` - Hook loading system
- `client/src/hooks/Dialog.js` - Dialog hook
- `client/src/hooks/Popover.js` - Popover hook
- `client/src/hooks/Sortable.js` - Sortable hook
- `client/src/hooks/Combobox.js` - Combobox hook
- `cmd/vango/registry/manifest.json` - Component registry
- `cmd/vango/registry/components/` - Component source files

### Modified Files
- `cmd/vango/commands/add.go` - Add command implementation
- `client/src/index.js` - Hook loader integration
- `client/src/event.js` - Hook event handling
- `pkg/vdom/hook.go` - Hook Go API

---

## Testing

### Component Testing

```go
func TestButton(t *testing.T) {
    // Default button
    node := ui.Button(ui.Children(ui.Text("Click")))
    html := vtest.Render(node)
    assert.Contains(t, html, "bg-primary")
    assert.Contains(t, html, "Click")
    
    // Destructive variant
    node = ui.Button(ui.Destructive, ui.Children(ui.Text("Delete")))
    html = vtest.Render(node)
    assert.Contains(t, html, "bg-destructive")
}
```

### Hook Testing

```javascript
// client/src/hooks/Dialog.test.js

describe('Dialog hook', () => {
    it('traps focus within dialog', () => {
        const el = createDialogElement();
        const hook = Dialog;
        const api = mockHookAPI();
        
        hook.mounted(el, {}, api);
        
        // Simulate Tab from last to first
        const event = new KeyboardEvent('keydown', { key: 'Tab' });
        el.dispatchEvent(event);
        
        expect(document.activeElement).toBe(el.querySelector('button'));
    });
    
    it('closes on Escape key', () => {
        const el = createDialogElement();
        const api = mockHookAPI();
        
        Dialog.mounted(el, {}, api);
        
        const event = new KeyboardEvent('keydown', { key: 'Escape' });
        el.dispatchEvent(event);
        
        expect(api.pushEvent).toHaveBeenCalledWith('close', {});
    });
});
```
