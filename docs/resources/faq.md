# FAQ

## General

**Q: Is this like Phoenix LiveView?**
A: Yes! Server-driven UI with binary patches over WebSocket, but for Go.

**Q: Do I need JavaScript?**
A: Only for Islands (third-party libraries) or custom Hooks.

**Q: SEO?**
A: Built-in SSR. Search engines see full HTML.

**Q: How does it scale?**
A: A single server handles thousands of concurrent users. Memory per session is typically 10-500KB.

## Technical

**Q: What if the connection drops?**
A: Auto-reconnect with state sync.

**Q: Can I use existing Go HTTP servers?**
A: Yes, Vango mounts as `http.Handler`.

**Q: Deployment?**
A: Single Go binary. No node_modules needed in production.

**Q: Hot reload?**
A: ~50ms rebuilds. Changes appear instantly.
