// Package toast provides feedback notifications for Vango applications.
//
// Since Vango uses persistent WebSocket connections, traditional HTTP
// flash cookies don't work. Instead, toasts use the generic ctx.Emit()
// mechanism to dispatch events to the client.
//
// # Zero Protocol Changes
//
// This package uses the existing Custom Event mechanism (ctx.Emit) rather
// than adding new protocol opcodes. This keeps the thin client at ~9.5KB
// and allows users to choose their own toast UI library.
//
// # Client-Side Handler
//
// The client-side handler is user-defined, allowing integration with
// any toast library (Toastify, Sonner, vanilla JS, etc.):
//
//	// user/app.js
//	window.addEventListener("vango:toast", (e) => {
//	    const { level, message, title } = e.detail;
//
//	    // Option 1: Use a library like Toastify
//	    Toastify({ text: message, className: level }).showToast();
//
//	    // Option 2: Use Sonner
//	    toast[level](message);
//
//	    // Option 3: Vanilla JS
//	    showCustomToast(level, message);
//	});
//
// # Server-Side Usage
//
// In your Vango handlers:
//
//	func DeleteProject(ctx vango.Ctx, id int) error {
//	    if err := db.Projects.Delete(id); err != nil {
//	        toast.Error(ctx, "Failed to delete project")
//	        return err
//	    }
//
//	    toast.Success(ctx, "Project deleted")
//	    ctx.Navigate("/projects")
//	    return nil
//	}
//
// With title:
//
//	toast.WithTitle(ctx, toast.TypeSuccess, "Settings", "Your changes have been saved.")
package toast
