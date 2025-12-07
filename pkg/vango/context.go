package vango

// SetContext sets a context value for the current component scope.
// This value will be available to all descendants via GetContext.
func SetContext(key, value any) {
	owner := getCurrentOwner()
	if owner != nil {
		owner.SetValue(key, value)
	}
}

// GetContext retrieves a context value from the nearest provider in the hierarchy.
// Returns nil if no value is found.
func GetContext(key any) any {
	owner := getCurrentOwner()
	if owner != nil {
		return owner.GetValue(key)
	}
	return nil
}

// SetValue sets a value on this Owner.
func (o *Owner) SetValue(key, value any) {
	o.valuesMu.Lock()
	defer o.valuesMu.Unlock()

	if o.values == nil {
		o.values = make(map[any]any)
	}
	o.values[key] = value
}

// GetValue retrieves a value from this Owner or its parents.
func (o *Owner) GetValue(key any) any {
	// Check self
	o.valuesMu.RLock()
	if o.values != nil {
		if val, ok := o.values[key]; ok {
			o.valuesMu.RUnlock()
			return val
		}
	}
	o.valuesMu.RUnlock()

	// Check parent
	if o.parent != nil {
		return o.parent.GetValue(key)
	}

	return nil
}
