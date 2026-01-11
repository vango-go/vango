package vango

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// Public Handler Types
// =============================================================================

// PageHandler is a function that renders a page.
// Two signatures are supported:
//   - func(ctx Ctx) *VNode                    - static page with no route params
//   - func(ctx Ctx, params P) *VNode          - dynamic page with typed params struct
//
// For dynamic pages, the params struct uses `param` tags to map route parameters:
//
//	type ShowParams struct {
//	    ID int `param:"id"`
//	}
//	func ShowPage(ctx vango.Ctx, p ShowParams) *vango.VNode { ... }
type PageHandler = any

// LayoutHandler wraps child content in a layout.
// Receives the render context and the slot containing child content.
//
//	func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
//	    return Html(
//	        Head(...),
//	        Body(children),
//	    )
//	}
type LayoutHandler = func(Ctx, Slot) *VNode

// APIHandler handles API requests and returns JSON responses.
// Multiple signatures are supported:
//   - func(ctx Ctx) (R, error)                        - no params or body
//   - func(ctx Ctx, params P) (R, error)              - with route params
//   - func(ctx Ctx, body B) (R, error)                - with request body
//   - func(ctx Ctx, params P, body B) (R, error)      - with both
//
// The framework inspects the handler signature to determine how to decode.
type APIHandler = any

// Slot represents child content passed to layouts.
// It is the rendered VNode tree of the wrapped page or nested layout.
type Slot = *VNode

// RouteMiddleware processes requests before they reach handlers.
type RouteMiddleware = router.Middleware

// =============================================================================
// Handler Wrappers
// =============================================================================

type apiRawBody struct {
	Bytes                 []byte
	ContentType           string
	StrictJSONContentType bool
}

func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	return contentType == "application/json" || strings.HasSuffix(contentType, "+json")
}

func decodeAPIRequestBody(raw any, targetType reflect.Type) (reflect.Value, error) {
	if targetType == nil {
		return reflect.Value{}, &HTTPError{
			Code:    http.StatusInternalServerError,
			Message: "invalid API handler: missing body type",
		}
	}

	valueOrZero := func(v any, t reflect.Type) reflect.Value {
		if v == nil {
			return reflect.Zero(t)
		}
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return reflect.Zero(t)
		}
		if rv.Type().AssignableTo(t) {
			return rv
		}
		if rv.Type().ConvertibleTo(t) {
			return rv.Convert(t)
		}
		return reflect.Zero(t)
	}

	// Fast path: raw already matches (or can be converted to) the target type.
	if raw != nil {
		rv := reflect.ValueOf(raw)
		if rv.IsValid() {
			if rv.Type().AssignableTo(targetType) {
				return rv, nil
			}
			if rv.Type().ConvertibleTo(targetType) {
				return rv.Convert(targetType), nil
			}
		}
	}

	var (
		bodyBytes    []byte
		contentType  string
		strictCTJSON bool
	)
	switch v := raw.(type) {
	case apiRawBody:
		bodyBytes = v.Bytes
		contentType = v.ContentType
		strictCTJSON = v.StrictJSONContentType
	case *apiRawBody:
		if v != nil {
			bodyBytes = v.Bytes
			contentType = v.ContentType
			strictCTJSON = v.StrictJSONContentType
		}
	case []byte:
		bodyBytes = v
	case json.RawMessage:
		bodyBytes = []byte(v)
	case string:
		bodyBytes = []byte(v)
	case nil:
		// keep zero values
	default:
		// Unknown input shape - treat as absent.
	}

	// Pass-through types.
	if targetType.Kind() == reflect.Slice && targetType.Elem().Kind() == reflect.Uint8 {
		return valueOrZero(bodyBytes, targetType), nil
	}
	if targetType.Kind() == reflect.String {
		return valueOrZero(string(bodyBytes), targetType), nil
	}

	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) == 0 {
		if targetType.Kind() == reflect.Pointer {
			return reflect.Zero(targetType), nil
		}
		return reflect.Value{}, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "missing request body",
		}
	}

	if (strictCTJSON || contentType != "") && !isJSONContentType(contentType) {
		return reflect.Value{}, &HTTPError{
			Code:    http.StatusUnsupportedMediaType,
			Message: "unsupported content type",
		}
	}

	// Decode JSON into the handler's declared body type.
	if targetType.Kind() == reflect.Pointer {
		// Allow explicit JSON null to map to nil.
		if bytes.Equal(trimmed, []byte("null")) {
			return reflect.Zero(targetType), nil
		}

		dst := reflect.New(targetType.Elem())
		if err := json.Unmarshal(trimmed, dst.Interface()); err != nil {
			return reflect.Value{}, &HTTPError{
				Code:    http.StatusBadRequest,
				Message: "invalid JSON body",
				Err:     err,
			}
		}
		return dst, nil
	}

	dst := reflect.New(targetType)
	if err := json.Unmarshal(trimmed, dst.Interface()); err != nil {
		return reflect.Value{}, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "invalid JSON body",
			Err:     err,
		}
	}
	return dst.Elem(), nil
}

// wrapPageHandler converts a user PageHandler to the internal router.PageHandler.
// It inspects the handler signature and creates an appropriate wrapper.
func wrapPageHandler(handler any) router.PageHandler {
	handlerVal := reflect.ValueOf(handler)
	handlerType := handlerVal.Type()

	valueOrZero := func(v any, t reflect.Type) reflect.Value {
		if v == nil {
			return reflect.Zero(t)
		}
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return reflect.Zero(t)
		}
		if rv.Type().AssignableTo(t) {
			return rv
		}
		if rv.Type().ConvertibleTo(t) {
			return rv.Convert(t)
		}
		return reflect.Zero(t)
	}

	// Validate it's a function
	if handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf("vango: page handler must be a function, got %T", handler))
	}

	numIn := handlerType.NumIn()
	numOut := handlerType.NumOut()

	// Validate output: must return *VNode
	if numOut != 1 {
		panic(fmt.Sprintf("vango: page handler must return exactly 1 value (*VNode), got %d", numOut))
	}

	switch numIn {
	case 1:
		// func(ctx Ctx) *VNode - static page
		// Type assert to the concrete function type
		fn, ok := handler.(func(Ctx) *VNode)
		if !ok {
			// Try reflection fallback for interface type
			return func(ctx server.Ctx, params any) vdom.Component {
				return vdom.Func(func() *vdom.VNode {
					results := handlerVal.Call([]reflect.Value{valueOrZero(ctx, handlerType.In(0))})
					return results[0].Interface().(*VNode)
				})
			}
		}
		return func(ctx server.Ctx, params any) vdom.Component {
			return vdom.Func(func() *vdom.VNode {
				return fn(ctx)
			})
		}

	case 2:
		// func(ctx Ctx, p P) *VNode - dynamic page with params
		paramsType := handlerType.In(1)
		decoder := buildParamDecoder(paramsType)

		return func(ctx server.Ctx, rawParams any) vdom.Component {
			// Decode params from map[string]string to typed struct
			paramsMap, ok := rawParams.(map[string]string)
			if !ok {
				paramsMap = make(map[string]string)
			}
			paramsVal := decoder(paramsMap) // stable across renders

			return vdom.Func(func() *vdom.VNode {
				// Call handler with decoded params during render so signals track properly.
				results := handlerVal.Call([]reflect.Value{
					valueOrZero(ctx, handlerType.In(0)),
					valueOrZero(paramsVal.Interface(), handlerType.In(1)),
				})
				return results[0].Interface().(*VNode)
			})
		}

	default:
		panic(fmt.Sprintf("vango: page handler has invalid signature (expected 1 or 2 args, got %d)", numIn))
	}
}

// wrapLayoutHandler converts a user LayoutHandler to the internal router.LayoutHandler.
func wrapLayoutHandler(handler LayoutHandler) router.LayoutHandler {
	return func(ctx server.Ctx, children router.Slot) *vdom.VNode {
		return handler(ctx, children)
	}
}

// wrapAPIHandler converts a user APIHandler to the internal router.APIHandler.
// It inspects the handler signature to determine how to decode params and body.
func wrapAPIHandler(handler any) router.APIHandler {
	handlerVal := reflect.ValueOf(handler)
	handlerType := handlerVal.Type()

	// Validate it's a function
	if handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf("vango: API handler must be a function, got %T", handler))
	}

	numIn := handlerType.NumIn()
	numOut := handlerType.NumOut()

	// Validate output: must return (R, error)
	if numOut != 2 {
		panic(fmt.Sprintf("vango: API handler must return exactly 2 values (result, error), got %d", numOut))
	}

	switch numIn {
	case 1:
		// func(ctx Ctx) (R, error) - no params or body
		return func(ctx server.Ctx, params any, body any) (any, error) {
			results := handlerVal.Call([]reflect.Value{reflect.ValueOf(ctx)})
			return extractAPIResults(results)
		}

	case 2:
		// func(ctx Ctx, params P) (R, error) OR func(ctx Ctx, body B) (R, error)
		// Heuristic: if second arg has `param` tag, it's params; otherwise body
		argType := handlerType.In(1)
		hasParamTags := hasParamStructTags(argType)

		if hasParamTags {
			// Params struct
			decoder := buildParamDecoder(argType)
			return func(ctx server.Ctx, rawParams any, body any) (any, error) {
				paramsMap, ok := rawParams.(map[string]string)
				if !ok {
					paramsMap = make(map[string]string)
				}
				paramsVal := decoder(paramsMap)
				results := handlerVal.Call([]reflect.Value{
					reflect.ValueOf(ctx),
					paramsVal,
				})
				return extractAPIResults(results)
			}
		} else {
			// Body struct
			return func(ctx server.Ctx, params any, rawBody any) (any, error) {
				bodyVal, err := decodeAPIRequestBody(rawBody, argType)
				if err != nil {
					return nil, err
				}
				results := handlerVal.Call([]reflect.Value{
					reflect.ValueOf(ctx),
					bodyVal,
				})
				return extractAPIResults(results)
			}
		}

	case 3:
		// func(ctx Ctx, params P, body B) (R, error) - both params and body
		paramsType := handlerType.In(1)
		decoder := buildParamDecoder(paramsType)

		return func(ctx server.Ctx, rawParams any, rawBody any) (any, error) {
			paramsMap, ok := rawParams.(map[string]string)
			if !ok {
				paramsMap = make(map[string]string)
			}
			paramsVal := decoder(paramsMap)

			bodyVal, err := decodeAPIRequestBody(rawBody, handlerType.In(2))
			if err != nil {
				return nil, err
			}

			results := handlerVal.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				paramsVal,
				bodyVal,
			})
			return extractAPIResults(results)
		}

	default:
		panic(fmt.Sprintf("vango: API handler has invalid signature (expected 1-3 args, got %d)", numIn))
	}
}

// extractAPIResults extracts the result and error from API handler reflection results.
func extractAPIResults(results []reflect.Value) (any, error) {
	result := results[0].Interface()
	errVal := results[1].Interface()
	if errVal != nil {
		return result, errVal.(error)
	}
	return result, nil
}

// =============================================================================
// Param Decoder
// =============================================================================

// paramFieldInfo holds pre-computed information for a struct field.
type paramFieldInfo struct {
	index     int          // Field index in struct
	paramName string       // Name from `param` tag
	fieldType reflect.Type // Field type
	converter func(string) (reflect.Value, error)
}

// buildParamDecoder creates a decoder function for a params struct type.
// It pre-computes field mappings at registration time for efficiency.
func buildParamDecoder(paramsType reflect.Type) func(map[string]string) reflect.Value {
	// Handle pointer types
	isPtr := paramsType.Kind() == reflect.Ptr
	if isPtr {
		paramsType = paramsType.Elem()
	}

	if paramsType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("vango: params type must be a struct, got %v", paramsType.Kind()))
	}

	// Pre-compute field info
	var fields []paramFieldInfo
	for i := 0; i < paramsType.NumField(); i++ {
		field := paramsType.Field(i)

		// Get param name from tag, fall back to lowercase field name
		paramName := field.Tag.Get("param")
		if paramName == "" {
			paramName = strings.ToLower(field.Name)
		}
		if paramName == "-" {
			continue // Skip fields tagged with "-"
		}

		converter := getTypeConverter(field.Type)
		fields = append(fields, paramFieldInfo{
			index:     i,
			paramName: paramName,
			fieldType: field.Type,
			converter: converter,
		})
	}

	return func(params map[string]string) reflect.Value {
		// Create new instance of params struct
		structVal := reflect.New(paramsType).Elem()

		// Populate fields from params map
		for _, fi := range fields {
			strVal, ok := params[fi.paramName]
			if !ok || strVal == "" {
				continue // Leave as zero value
			}

			converted, err := fi.converter(strVal)
			if err != nil {
				// Invalid param value - leave as zero value
				// In production, this would be handled by validation middleware
				continue
			}
			structVal.Field(fi.index).Set(converted)
		}

		if isPtr {
			ptr := reflect.New(paramsType)
			ptr.Elem().Set(structVal)
			return ptr
		}
		return structVal
	}
}

// getTypeConverter returns a converter function for the given type.
func getTypeConverter(t reflect.Type) func(string) (reflect.Value, error) {
	if t.Kind() == reflect.Ptr {
		elem := t.Elem()
		elemConv := getTypeConverter(elem)
		return func(s string) (reflect.Value, error) {
			val, err := elemConv(s)
			if err != nil {
				return reflect.Value{}, err
			}
			ptr := reflect.New(elem)
			if val.IsValid() {
				if val.Type().AssignableTo(elem) {
					ptr.Elem().Set(val)
				} else if val.Type().ConvertibleTo(elem) {
					ptr.Elem().Set(val.Convert(elem))
				} else {
					return reflect.Value{}, fmt.Errorf("cannot convert %v to %v", val.Type(), elem)
				}
			}
			return ptr, nil
		}
	}

	textUnmarshaler := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	if reflect.PointerTo(t).Implements(textUnmarshaler) {
		return func(s string) (reflect.Value, error) {
			ptr := reflect.New(t)
			u := ptr.Interface().(encoding.TextUnmarshaler)
			if err := u.UnmarshalText([]byte(s)); err != nil {
				return reflect.Value{}, err
			}
			return ptr.Elem(), nil
		}
	}
	if t.Implements(textUnmarshaler) {
		return func(s string) (reflect.Value, error) {
			val := reflect.New(t).Elem()
			u := val.Interface().(encoding.TextUnmarshaler)
			if err := u.UnmarshalText([]byte(s)); err != nil {
				return reflect.Value{}, err
			}
			return val, nil
		}
	}

	switch t.Kind() {
	case reflect.String:
		return func(s string) (reflect.Value, error) {
			v := reflect.ValueOf(s)
			if v.Type().AssignableTo(t) {
				return v, nil
			}
			if v.Type().ConvertibleTo(t) {
				return v.Convert(t), nil
			}
			return reflect.Value{}, fmt.Errorf("cannot convert %v to %v", v.Type(), t)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(s string) (reflect.Value, error) {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(n).Convert(t), nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(s string) (reflect.Value, error) {
			n, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(n).Convert(t), nil
		}

	case reflect.Float32, reflect.Float64:
		return func(s string) (reflect.Value, error) {
			n, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(n).Convert(t), nil
		}

	case reflect.Bool:
		return func(s string) (reflect.Value, error) {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return reflect.Value{}, err
			}
			v := reflect.ValueOf(b)
			if v.Type().AssignableTo(t) {
				return v, nil
			}
			if v.Type().ConvertibleTo(t) {
				return v.Convert(t), nil
			}
			return reflect.Value{}, fmt.Errorf("cannot convert %v to %v", v.Type(), t)
		}

	default:
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.String {
			return func(s string) (reflect.Value, error) {
				var parts []string
				if s != "" {
					parts = strings.Split(s, "/")
				}
				v := reflect.ValueOf(parts)
				if v.Type().AssignableTo(t) {
					return v, nil
				}
				if v.Type().ConvertibleTo(t) {
					return v.Convert(t), nil
				}
				return reflect.Value{}, fmt.Errorf("cannot convert %v to %v", v.Type(), t)
			}
		}

		// Unsupported type; leave as zero value.
		return func(s string) (reflect.Value, error) {
			return reflect.Value{}, fmt.Errorf("unsupported param field type: %v", t)
		}
	}
}

// hasParamStructTags checks if a type has any `param` struct tags.
func hasParamStructTags(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("param") != "" {
			return true
		}
	}
	return false
}
