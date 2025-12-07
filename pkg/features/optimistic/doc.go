// Package optimistic provides instant visual feedback for user interactions.
//
// Optimistic updates allow the thin client to immediately apply visual changes
// when a user interacts with an element, before the server roundtrip completes.
// This creates a perceived latency of 0ms for common interactions like button
// clicks, toggles, and form submissions.
//
// # How It Works
//
// When an element has optimistic attributes, the thin client:
//  1. Captures the user interaction (click, input, etc.)
//  2. Immediately applies the optimistic changes to the DOM
//  3. Sends the event to the server
//  4. When server patches arrive, they replace the optimistic state
//  5. If the server response differs, the optimistic change is reverted
//
// # Available Functions
//
//   - OptimisticClass: Toggle a CSS class immediately on click
//   - OptimisticText: Change text content immediately
//   - OptimisticAttr: Change an attribute value immediately
//   - OptimisticParentClass: Toggle a class on the parent element
//   - OptimisticDisable: Disable the element immediately (prevent double-clicks)
//   - OptimisticHide: Hide the element immediately
//
// # Example Usage
//
//	// Like button with instant feedback
//	func LikeButton(postID string, likes int, userLiked bool) *vdom.VNode {
//	    return Button(
//	        Class("like-button"),
//	        ClassIf(userLiked, "liked"),
//	        OnClick(func() {
//	            if userLiked {
//	                db.Posts.Unlike(postID)
//	            } else {
//	                db.Posts.Like(postID)
//	            }
//	        }),
//	        // Instant visual feedback
//	        OptimisticClass("liked", !userLiked),
//	        OptimisticText(fmt.Sprintf("%d", likes + boolToInt(!userLiked))),
//	        Icon("heart"),
//	        Span(Textf("%d", likes)),
//	    )
//	}
//
// # Wire Format
//
// Optimistic attributes are rendered as data-optimistic-* attributes:
//
//	<button data-hid="h5"
//	        data-optimistic-class="liked:add"
//	        data-optimistic-text="43">
//	    <span>42</span>
//	</button>
//
// The thin client parses these and applies them immediately on interaction.
//
// # Revert Behavior
//
// If the server's response differs from the optimistic prediction (e.g., the
// like failed due to rate limiting), the client automatically reverts to the
// server-provided state. This ensures eventual consistency.
package optimistic
