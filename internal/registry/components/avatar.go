//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// AvatarOption configures an Avatar component.
type AvatarOption func(*avatarConfig)

type avatarConfig struct {
	src       string
	alt       string
	fallback  string
	size      Size
	className string
}

func defaultAvatarConfig() avatarConfig {
	return avatarConfig{
		size: SizeMd,
	}
}

// AvatarSrc sets the image source.
func AvatarSrc(src string) AvatarOption {
	return func(c *avatarConfig) {
		c.src = src
	}
}

// AvatarAlt sets the alt text.
func AvatarAlt(alt string) AvatarOption {
	return func(c *avatarConfig) {
		c.alt = alt
	}
}

// AvatarFallback sets the fallback text (typically initials).
func AvatarFallback(fallback string) AvatarOption {
	return func(c *avatarConfig) {
		c.fallback = fallback
	}
}

// AvatarSize sets the avatar size.
func AvatarSize(size Size) AvatarOption {
	return func(c *avatarConfig) {
		c.size = size
	}
}

// AvatarSm sets the avatar to small size.
func AvatarSm() AvatarOption {
	return func(c *avatarConfig) {
		c.size = SizeSm
	}
}

// AvatarLg sets the avatar to large size.
func AvatarLg() AvatarOption {
	return func(c *avatarConfig) {
		c.size = SizeLg
	}
}

// AvatarClass adds additional CSS classes.
func AvatarClass(className string) AvatarOption {
	return func(c *avatarConfig) {
		c.className = className
	}
}

// Avatar renders an avatar component.
func Avatar(opts ...AvatarOption) *VNode {
	cfg := defaultAvatarConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Size classes
	sizeClasses := map[Size]string{
		SizeXS: "h-6 w-6 text-xs",
		SizeSm: "h-8 w-8 text-sm",
		SizeMd: "h-10 w-10 text-sm",
		SizeLg: "h-12 w-12 text-base",
		SizeXL: "h-16 w-16 text-lg",
	}

	containerClasses := CN(
		"relative flex shrink-0 overflow-hidden rounded-full",
		sizeClasses[cfg.size],
		cfg.className,
	)

	// Image element (will be hidden if src fails)
	var imgEl *VNode
	if cfg.src != "" {
		imgEl = El("img",
			Class("aspect-square h-full w-full"),
			Src(cfg.src),
			Alt(cfg.alt),
		)
	}

	// Fallback element
	fallbackEl := Div(
		Class("flex h-full w-full items-center justify-center rounded-full bg-muted text-muted-foreground"),
		Text(cfg.fallback),
	)

	// If we have a src, show image with fallback via CSS (onError handling would be client-side)
	if cfg.src != "" {
		return Div(
			Class(containerClasses),
			imgEl,
		)
	}

	// No src, show fallback directly
	return Div(
		Class(containerClasses),
		fallbackEl,
	)
}

// AvatarGroup renders a group of overlapping avatars.
func AvatarGroup(avatars ...*VNode) *VNode {
	children := make([]any, len(avatars))
	for i, avatar := range avatars {
		// Add negative margin for overlap (except first)
		if i > 0 {
			children[i] = Div(
				Class("-ml-4"),
				avatar,
			)
		} else {
			children[i] = avatar
		}
	}

	return Div(
		Class("flex items-center"),
		children...,
	)
}
