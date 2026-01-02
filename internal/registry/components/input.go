package ui

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// InputOption configures an Input component.
type InputOption func(*inputConfig)

type inputConfig struct {
	inputType   string
	name        string
	placeholder string
	value       string
	disabled    bool
	required    bool
	readonly    bool
	className   string
	onInput     func(string)
	onChange    func(string)
	onFocus     func()
	onBlur      func()
	attrs       map[string]string
}

func defaultInputConfig() inputConfig {
	return inputConfig{
		inputType: "text",
	}
}

// InputType sets the input type (text, email, password, etc.).
func InputType(t string) InputOption {
	return func(c *inputConfig) {
		c.inputType = t
	}
}

// InputName sets the input name attribute.
func InputName(name string) InputOption {
	return func(c *inputConfig) {
		c.name = name
	}
}

// InputPlaceholder sets the placeholder text.
func InputPlaceholder(placeholder string) InputOption {
	return func(c *inputConfig) {
		c.placeholder = placeholder
	}
}

// InputValue sets the input value.
func InputValue(value string) InputOption {
	return func(c *inputConfig) {
		c.value = value
	}
}

// InputDisabled sets the disabled state.
func InputDisabled(disabled bool) InputOption {
	return func(c *inputConfig) {
		c.disabled = disabled
	}
}

// InputRequired sets the required state.
func InputRequired(required bool) InputOption {
	return func(c *inputConfig) {
		c.required = required
	}
}

// InputReadonly sets the readonly state.
func InputReadonly(readonly bool) InputOption {
	return func(c *inputConfig) {
		c.readonly = readonly
	}
}

// InputClass adds additional CSS classes.
func InputClass(className string) InputOption {
	return func(c *inputConfig) {
		c.className = className
	}
}

// InputOnInput sets the input event handler.
func InputOnInput(handler func(string)) InputOption {
	return func(c *inputConfig) {
		c.onInput = handler
	}
}

// InputOnChange sets the change event handler.
func InputOnChange(handler func(string)) InputOption {
	return func(c *inputConfig) {
		c.onChange = handler
	}
}

// InputOnFocus sets the focus event handler.
func InputOnFocus(handler func()) InputOption {
	return func(c *inputConfig) {
		c.onFocus = handler
	}
}

// InputOnBlur sets the blur event handler.
func InputOnBlur(handler func()) InputOption {
	return func(c *inputConfig) {
		c.onBlur = handler
	}
}

// InputAttr adds a custom attribute.
func InputAttr(name, value string) InputOption {
	return func(c *inputConfig) {
		if c.attrs == nil {
			c.attrs = make(map[string]string)
		}
		c.attrs[name] = value
	}
}

// Input renders an input element.
func Input(opts ...InputOption) *VNode {
	cfg := defaultInputConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
		cfg.className,
	)

	attrs := []any{
		Class(classes),
		Type(cfg.inputType),
	}

	if cfg.name != "" {
		attrs = append(attrs, Name(cfg.name))
	}
	if cfg.placeholder != "" {
		attrs = append(attrs, Placeholder(cfg.placeholder))
	}
	if cfg.value != "" {
		attrs = append(attrs, Value(cfg.value))
	}
	if cfg.disabled {
		attrs = append(attrs, Disabled())
	}
	if cfg.required {
		attrs = append(attrs, Required())
	}
	if cfg.readonly {
		attrs = append(attrs, Readonly())
	}
	if cfg.onInput != nil {
		attrs = append(attrs, OnInput(cfg.onInput))
	}
	if cfg.onChange != nil {
		attrs = append(attrs, OnChange(cfg.onChange))
	}
	if cfg.onFocus != nil {
		attrs = append(attrs, OnFocus(cfg.onFocus))
	}
	if cfg.onBlur != nil {
		attrs = append(attrs, OnBlur(cfg.onBlur))
	}

	for name, value := range cfg.attrs {
		attrs = append(attrs, Attr{Key: name, Value: value})
	}

	return El("input", attrs...)
}

// EmailInput is a convenience wrapper for email inputs.
func EmailInput(opts ...InputOption) *VNode {
	return Input(append([]InputOption{InputType("email")}, opts...)...)
}

// PasswordInput is a convenience wrapper for password inputs.
func PasswordInput(opts ...InputOption) *VNode {
	return Input(append([]InputOption{InputType("password")}, opts...)...)
}

// SearchInput is a convenience wrapper for search inputs.
func SearchInput(opts ...InputOption) *VNode {
	return Input(append([]InputOption{InputType("search")}, opts...)...)
}

// NumberInput is a convenience wrapper for number inputs.
func NumberInput(opts ...InputOption) *VNode {
	return Input(append([]InputOption{InputType("number")}, opts...)...)
}
