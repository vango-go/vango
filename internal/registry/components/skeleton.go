//go:build vangoui

package ui

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// SkeletonOption configures a Skeleton component.
type SkeletonOption func(*skeletonConfig)

type skeletonConfig struct {
	className string
}

// SkeletonClass adds additional CSS classes.
func SkeletonClass(className string) SkeletonOption {
	return func(c *skeletonConfig) {
		c.className = className
	}
}

// Skeleton renders a loading skeleton placeholder.
func Skeleton(opts ...SkeletonOption) *VNode {
	cfg := skeletonConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"animate-pulse rounded-md bg-muted",
		cfg.className,
	)

	return Div(Class(classes))
}

// SkeletonText renders a text-sized skeleton.
func SkeletonText(opts ...SkeletonOption) *VNode {
	return Skeleton(append(opts, SkeletonClass("h-4 w-full"))...)
}

// SkeletonTitle renders a title-sized skeleton.
func SkeletonTitle(opts ...SkeletonOption) *VNode {
	return Skeleton(append(opts, SkeletonClass("h-6 w-3/4"))...)
}

// SkeletonCircle renders a circular skeleton (for avatars).
func SkeletonCircle(opts ...SkeletonOption) *VNode {
	return Skeleton(append(opts, SkeletonClass("h-12 w-12 rounded-full"))...)
}

// SkeletonCard renders a card-shaped skeleton.
func SkeletonCard(opts ...SkeletonOption) *VNode {
	return Div(
		Class("space-y-3"),
		Skeleton(SkeletonClass("h-40 w-full rounded-lg")),
		Skeleton(SkeletonClass("h-4 w-3/4")),
		Skeleton(SkeletonClass("h-4 w-1/2")),
	)
}
