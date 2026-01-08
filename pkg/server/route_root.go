package server

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

type routeRootComponent struct {
	session   *Session
	page      Component
	layouts   []LayoutHandler
	canonPath string
	query     string
	params    map[string]string
}

func (c *routeRootComponent) Render() *vdom.VNode {
	var renderCtx Ctx
	if v := vango.UseCtx(); v != nil {
		if got, ok := v.(Ctx); ok {
			renderCtx = got
		}
	}
	if renderCtx == nil && c.session != nil {
		renderCtx = c.session.createRenderContext()
	}

	if ctxImpl, ok := renderCtx.(*ctx); ok {
		ctxImpl.setParams(c.params)
		if ctxImpl.request == nil {
			ctxImpl.request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path:     c.canonPath,
					RawQuery: c.query,
				},
			}
		}
	}

	// Ensure ctx.Path() reflects the current route during server-driven rendering.
	if c.session != nil {
		c.session.CurrentRoute = c.canonPath
	}

	result := c.page.Render()

	for i := len(c.layouts) - 1; i >= 0; i-- {
		result = c.layouts[i](renderCtx, result)
	}

	return result
}

func newRouteRootComponent(session *Session, router Router, fullPath string) (Component, string, error) {
	if router == nil {
		return nil, "", errors.New("router is nil")
	}

	canonPath, query, _, err := CanonicalizePath(fullPath)
	if err != nil {
		return nil, "", err
	}

	match, ok := router.Match("GET", canonPath)
	if !ok {
		notFound := router.NotFound()
		if notFound == nil {
			return nil, "", errors.New("route not found")
		}
		match = &simpleRouteMatch{
			pageHandler: notFound,
			params:      map[string]string{},
		}
	}

	params := match.GetParams()
	pageHandler := match.GetPageHandler()
	if pageHandler == nil {
		return nil, "", errors.New("page handler is nil")
	}

	if session != nil {
		session.CurrentRoute = canonPath
	}

	// Create the page component once so it has stable identity across rerenders.
	renderCtx := session.createRenderContext()
	if ctxImpl, ok := renderCtx.(*ctx); ok {
		ctxImpl.setParams(params)
		ctxImpl.request = &http.Request{
			Method: http.MethodGet,
			URL: &url.URL{
				Path:     canonPath,
				RawQuery: query,
			},
		}
	}
	page := pageHandler(renderCtx, params)
	if page == nil {
		return nil, "", errors.New("page handler returned nil component")
	}

	return &routeRootComponent{
		session:   session,
		page:      page,
		layouts:   match.GetLayoutHandlers(),
		canonPath: canonPath,
		query:     query,
		params:    params,
	}, canonPath, nil
}
