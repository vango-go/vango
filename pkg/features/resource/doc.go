// Package resource provides async data loading and management for Vango applications.
//
// Resources are reactive primitives that handle the complete lifecycle of asynchronous
// data fetching, including:
//
//   - Loading, Error, and Success states
//   - Automatic dependency tracking and re-fetching
//   - Caching and stale time management
//   - Optimistic updates and mutations
//   - Pattern matching for UI rendering
//
// Basic Usage:
//
//	user := resource.New(func() (*User, error) {
//	    return db.Users.Find(id)
//	})
//
//	return user.Match(
//	    resource.OnLoading(func() *vdom.VNode { return Loading() }),
//	    resource.OnError(func(err error) *vdom.VNode { return Error(err) }),
//	    resource.OnReady(func(u *User) *vdom.VNode { return UserProfile(u) }),
//	)
package resource
