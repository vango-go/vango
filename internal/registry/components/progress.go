//go:build vangoui

package ui

import (
	"fmt"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// ProgressOption configures a Progress component.
type ProgressOption func(*progressConfig)

type progressConfig struct {
	value     int
	max       int
	className string
	showLabel bool
}

func defaultProgressConfig() progressConfig {
	return progressConfig{
		value: 0,
		max:   100,
	}
}

// ProgressValue sets the current progress value.
func ProgressValue(value int) ProgressOption {
	return func(c *progressConfig) {
		c.value = value
	}
}

// ProgressMax sets the maximum value.
func ProgressMax(max int) ProgressOption {
	return func(c *progressConfig) {
		c.max = max
	}
}

// ProgressClass adds additional CSS classes.
func ProgressClass(className string) ProgressOption {
	return func(c *progressConfig) {
		c.className = className
	}
}

// ProgressShowLabel shows the percentage label.
func ProgressShowLabel(show bool) ProgressOption {
	return func(c *progressConfig) {
		c.showLabel = show
	}
}

// Progress renders a progress bar.
func Progress(opts ...ProgressOption) *VNode {
	cfg := defaultProgressConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Calculate percentage
	percentage := 0
	if cfg.max > 0 {
		percentage = (cfg.value * 100) / cfg.max
	}
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	containerClasses := CN(
		"relative h-4 w-full overflow-hidden rounded-full bg-secondary",
		cfg.className,
	)

	indicatorStyle := fmt.Sprintf("width: %d%%", percentage)

	container := Div(
		Class(containerClasses),
		Role("progressbar"),
		AriaValueNow(float64(cfg.value)),
		AriaValueMin(0),
		AriaValueMax(float64(cfg.max)),
		Div(
			Class("h-full w-full flex-1 bg-primary transition-all"),
			StyleAttr(indicatorStyle),
		),
	)

	if cfg.showLabel {
		return Div(
			Class("space-y-1"),
			Div(
				Class("flex justify-between text-sm"),
				El("span", Text("Progress")),
				El("span", Text(fmt.Sprintf("%d%%", percentage))),
			),
			container,
		)
	}

	return container
}
