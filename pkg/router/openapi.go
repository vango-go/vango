package router

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// OpenAPIGenerator generates OpenAPI 3.0 specifications from API routes.
type OpenAPIGenerator struct {
	routesDir  string
	modulePath string
	info       OpenAPIInfo
}

// OpenAPIInfo contains API metadata.
type OpenAPIInfo struct {
	Title       string
	Description string
	Version     string
}

// NewOpenAPIGenerator creates a new OpenAPI generator.
func NewOpenAPIGenerator(routesDir, modulePath string, info OpenAPIInfo) *OpenAPIGenerator {
	if info.Title == "" {
		info.Title = "API"
	}
	if info.Version == "" {
		info.Version = "1.0.0"
	}
	return &OpenAPIGenerator{
		routesDir:  routesDir,
		modulePath: modulePath,
		info:       info,
	}
}

// OpenAPISpec represents an OpenAPI 3.0 specification.
type OpenAPISpec struct {
	OpenAPI    string                    `json:"openapi"`
	Info       OpenAPISpecInfo           `json:"info"`
	Paths      map[string]OpenAPIPath    `json:"paths"`
	Components *OpenAPIComponents        `json:"components,omitempty"`
}

// OpenAPISpecInfo contains API info.
type OpenAPISpecInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// OpenAPIPath represents path operations.
type OpenAPIPath map[string]*OpenAPIOperation

// OpenAPIOperation represents an HTTP operation.
type OpenAPIOperation struct {
	Summary     string                      `json:"summary,omitempty"`
	Description string                      `json:"description,omitempty"`
	OperationID string                      `json:"operationId,omitempty"`
	Tags        []string                    `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter          `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse  `json:"responses"`
}

// OpenAPIParameter represents a request parameter.
type OpenAPIParameter struct {
	Name        string            `json:"name"`
	In          string            `json:"in"` // path, query, header
	Description string            `json:"description,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Schema      *OpenAPISchema    `json:"schema"`
}

// OpenAPIRequestBody represents a request body.
type OpenAPIRequestBody struct {
	Description string                     `json:"description,omitempty"`
	Required    bool                       `json:"required,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content"`
}

// OpenAPIResponse represents a response.
type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

// OpenAPIMediaType represents media type content.
type OpenAPIMediaType struct {
	Schema *OpenAPISchema `json:"schema,omitempty"`
}

// OpenAPISchema represents a JSON schema.
type OpenAPISchema struct {
	Type       string                   `json:"type,omitempty"`
	Format     string                   `json:"format,omitempty"`
	Items      *OpenAPISchema           `json:"items,omitempty"`
	Properties map[string]*OpenAPISchema `json:"properties,omitempty"`
	Required   []string                 `json:"required,omitempty"`
	Ref        string                   `json:"$ref,omitempty"`
}

// OpenAPIComponents contains reusable components.
type OpenAPIComponents struct {
	Schemas map[string]*OpenAPISchema `json:"schemas,omitempty"`
}

// APIEndpoint represents a discovered API endpoint.
type APIEndpoint struct {
	Path        string
	Method      string
	FuncName    string
	Package     string
	Description string
	Params      []ParamDef
	RequestType *TypeInfo
	ResponseType *TypeInfo
}

// TypeInfo contains information about a Go type.
type TypeInfo struct {
	Name       string
	Package    string
	Fields     []FieldInfo
	IsSlice    bool
	IsPointer  bool
}

// FieldInfo contains information about a struct field.
type FieldInfo struct {
	Name     string
	Type     string
	JSONName string
	Required bool
}

// Generate creates an OpenAPI specification from the API routes.
func (g *OpenAPIGenerator) Generate() ([]byte, error) {
	// Find API routes directory
	apiDir := filepath.Join(g.routesDir, "api")

	// Scan for API endpoints
	endpoints, types, err := g.scanAPIRoutes(apiDir)
	if err != nil {
		return nil, err
	}

	// Build OpenAPI spec
	spec := OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPISpecInfo{
			Title:       g.info.Title,
			Description: g.info.Description,
			Version:     g.info.Version,
		},
		Paths:      make(map[string]OpenAPIPath),
		Components: &OpenAPIComponents{
			Schemas: make(map[string]*OpenAPISchema),
		},
	}

	// Add schemas for types
	for name, typeInfo := range types {
		spec.Components.Schemas[name] = g.typeToSchema(typeInfo)
	}

	// Group endpoints by path
	pathEndpoints := make(map[string][]APIEndpoint)
	for _, ep := range endpoints {
		pathEndpoints[ep.Path] = append(pathEndpoints[ep.Path], ep)
	}

	// Convert endpoints to OpenAPI paths
	for path, eps := range pathEndpoints {
		pathItem := make(OpenAPIPath)
		for _, ep := range eps {
			op := g.endpointToOperation(ep)
			pathItem[strings.ToLower(ep.Method)] = op
		}
		spec.Paths[path] = pathItem
	}

	// Sort paths for deterministic output
	sortedPaths := make(map[string]OpenAPIPath)
	var pathKeys []string
	for k := range spec.Paths {
		pathKeys = append(pathKeys, k)
	}
	sort.Strings(pathKeys)
	for _, k := range pathKeys {
		sortedPaths[k] = spec.Paths[k]
	}
	spec.Paths = sortedPaths

	// Marshal to JSON
	return json.MarshalIndent(spec, "", "  ")
}

// scanAPIRoutes scans the API directory for route definitions.
func (g *OpenAPIGenerator) scanAPIRoutes(apiDir string) ([]APIEndpoint, map[string]*TypeInfo, error) {
	var endpoints []APIEndpoint
	types := make(map[string]*TypeInfo)

	err := filepath.WalkDir(apiDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		eps, typeInfos, err := g.parseAPIFile(path, apiDir)
		if err != nil {
			return err
		}

		endpoints = append(endpoints, eps...)
		for name, info := range typeInfos {
			types[name] = info
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return endpoints, types, nil
}

// parseAPIFile parses a Go file for API handler definitions.
func (g *OpenAPIGenerator) parseAPIFile(path, apiDir string) ([]APIEndpoint, map[string]*TypeInfo, error) {
	var endpoints []APIEndpoint
	types := make(map[string]*TypeInfo)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	// Determine URL path from file path
	relPath, _ := filepath.Rel(apiDir, path)
	urlPath := g.fileToURLPath(relPath)

	// Extract params from path
	params := extractParamsFromURLPath(urlPath)

	// Scan for type definitions
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			typeInfo := &TypeInfo{
				Name:    typeSpec.Name.Name,
				Package: f.Name.Name,
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}

				fieldInfo := FieldInfo{
					Name:     field.Names[0].Name,
					Type:     g.typeExprToString(field.Type),
					JSONName: g.getJSONTag(field),
					Required: g.hasValidateRequired(field),
				}
				typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
			}

			types[typeSpec.Name.Name] = typeInfo
		}
	}

	// Scan for handler functions
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}

		// Check for HTTP method suffix (GET, POST, PUT, DELETE, etc.)
		method := extractHTTPMethod(fn.Name.Name)
		if method == "" {
			continue
		}

		ep := APIEndpoint{
			Path:     urlPath,
			Method:   method,
			FuncName: fn.Name.Name,
			Package:  f.Name.Name,
			Params:   params,
		}

		// Extract description from doc comment
		if fn.Doc != nil {
			ep.Description = strings.TrimSpace(fn.Doc.Text())
		}

		// Analyze function parameters for request type
		if fn.Type.Params != nil {
			for _, param := range fn.Type.Params.List {
				typeName := g.typeExprToString(param.Type)
				// Skip common parameters (ctx, Params)
				if strings.Contains(typeName, "Ctx") || strings.Contains(typeName, "Params") {
					continue
				}
				// This is likely the request body type
				ep.RequestType = &TypeInfo{
					Name:      typeName,
					IsPointer: strings.HasPrefix(typeName, "*"),
				}
				break
			}
		}

		// Analyze function return types for response type
		if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
			for _, result := range fn.Type.Results.List {
				typeName := g.typeExprToString(result.Type)
				// Skip error type
				if typeName == "error" {
					continue
				}
				// Skip Response wrapper, extract inner type
				if strings.HasPrefix(typeName, "vango.Response[") {
					inner := strings.TrimPrefix(typeName, "vango.Response[")
					inner = strings.TrimSuffix(inner, "]")
					typeName = inner
				}
				ep.ResponseType = &TypeInfo{
					Name:      strings.TrimPrefix(typeName, "*"),
					IsPointer: strings.HasPrefix(typeName, "*"),
					IsSlice:   strings.HasPrefix(typeName, "[]"),
				}
				break
			}
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints, types, nil
}

// fileToURLPath converts a file path to a URL path.
func (g *OpenAPIGenerator) fileToURLPath(relPath string) string {
	// Remove .go extension
	path := strings.TrimSuffix(relPath, ".go")

	// Convert Windows path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Handle index files
	if strings.HasSuffix(path, "/index") {
		path = strings.TrimSuffix(path, "/index")
	}
	if path == "index" {
		path = ""
	}

	// Convert [param] to {param} for OpenAPI
	path = convertParamsToOpenAPI(path)

	// Add /api prefix
	if path == "" {
		return "/api"
	}
	return "/api/" + path
}

// convertParamsToOpenAPI converts [param] notation to {param}.
func convertParamsToOpenAPI(path string) string {
	// [id] → {id}
	// [id:int] → {id}
	result := path
	result = strings.ReplaceAll(result, "[", "{")
	result = strings.ReplaceAll(result, "]", "}")

	// Remove type annotations
	for strings.Contains(result, ":") {
		start := strings.Index(result, "{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start

		inner := result[start+1 : end]
		if idx := strings.Index(inner, ":"); idx != -1 {
			inner = inner[:idx]
		}
		result = result[:start+1] + inner + result[end:]
	}

	return result
}

// extractParamsFromURLPath extracts parameters from a URL path.
func extractParamsFromURLPath(path string) []ParamDef {
	var params []ParamDef

	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := part[1 : len(part)-1]
			params = append(params, ParamDef{
				Name:    name,
				Type:    inferParamTypeFromName(name),
				Segment: part,
			})
		}
	}

	return params
}

// extractHTTPMethod extracts the HTTP method from a function name.
func extractHTTPMethod(funcName string) string {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	for _, method := range methods {
		if strings.HasSuffix(funcName, method) {
			return method
		}
	}
	return ""
}

// endpointToOperation converts an API endpoint to an OpenAPI operation.
func (g *OpenAPIGenerator) endpointToOperation(ep APIEndpoint) *OpenAPIOperation {
	op := &OpenAPIOperation{
		OperationID: ep.FuncName,
		Description: ep.Description,
		Tags:        []string{ep.Package},
		Responses: map[string]OpenAPIResponse{
			"200": {Description: "Successful response"},
		},
	}

	// Add path parameters
	for _, param := range ep.Params {
		op.Parameters = append(op.Parameters, OpenAPIParameter{
			Name:     param.Name,
			In:       "path",
			Required: true,
			Schema: &OpenAPISchema{
				Type: g.goTypeToOpenAPIType(param.Type),
			},
		})
	}

	// Add request body for POST, PUT, PATCH
	if (ep.Method == "POST" || ep.Method == "PUT" || ep.Method == "PATCH") && ep.RequestType != nil {
		typeName := strings.TrimPrefix(ep.RequestType.Name, "*")
		op.RequestBody = &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Ref: "#/components/schemas/" + typeName,
					},
				},
			},
		}
	}

	// Add response schema
	if ep.ResponseType != nil {
		typeName := ep.ResponseType.Name
		if ep.ResponseType.IsSlice {
			typeName = strings.TrimPrefix(typeName, "[]")
			typeName = strings.TrimPrefix(typeName, "*")
			op.Responses["200"] = OpenAPIResponse{
				Description: "Successful response",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: &OpenAPISchema{
							Type: "array",
							Items: &OpenAPISchema{
								Ref: "#/components/schemas/" + typeName,
							},
						},
					},
				},
			}
		} else {
			typeName = strings.TrimPrefix(typeName, "*")
			op.Responses["200"] = OpenAPIResponse{
				Description: "Successful response",
				Content: map[string]OpenAPIMediaType{
					"application/json": {
						Schema: &OpenAPISchema{
							Ref: "#/components/schemas/" + typeName,
						},
					},
				},
			}
		}
	}

	return op
}

// typeToSchema converts a TypeInfo to an OpenAPI schema.
func (g *OpenAPIGenerator) typeToSchema(info *TypeInfo) *OpenAPISchema {
	schema := &OpenAPISchema{
		Type:       "object",
		Properties: make(map[string]*OpenAPISchema),
	}

	for _, field := range info.Fields {
		jsonName := field.JSONName
		if jsonName == "" || jsonName == "-" {
			continue
		}

		propSchema := &OpenAPISchema{
			Type: g.goTypeToOpenAPIType(field.Type),
		}

		// Handle slices
		if strings.HasPrefix(field.Type, "[]") {
			propSchema.Type = "array"
			itemType := strings.TrimPrefix(field.Type, "[]")
			itemType = strings.TrimPrefix(itemType, "*")
			propSchema.Items = &OpenAPISchema{
				Type: g.goTypeToOpenAPIType(itemType),
			}
		}

		schema.Properties[jsonName] = propSchema

		if field.Required {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema
}

// goTypeToOpenAPIType converts a Go type to an OpenAPI type.
func (g *OpenAPIGenerator) goTypeToOpenAPIType(goType string) string {
	goType = strings.TrimPrefix(goType, "*")

	switch goType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		if strings.HasPrefix(goType, "[]") {
			return "array"
		}
		return "object"
	}
}

// typeExprToString converts an ast.Expr type to a string representation.
func (g *OpenAPIGenerator) typeExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + g.typeExprToString(t.X)
	case *ast.ArrayType:
		return "[]" + g.typeExprToString(t.Elt)
	case *ast.SelectorExpr:
		return g.typeExprToString(t.X) + "." + t.Sel.Name
	case *ast.IndexExpr:
		return g.typeExprToString(t.X) + "[" + g.typeExprToString(t.Index) + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// getJSONTag extracts the JSON field name from a struct tag.
func (g *OpenAPIGenerator) getJSONTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}

	tag := strings.Trim(field.Tag.Value, "`")
	parts := strings.Split(tag, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "json:") {
			value := strings.TrimPrefix(part, "json:")
			value = strings.Trim(value, "\"")
			if idx := strings.Index(value, ","); idx != -1 {
				value = value[:idx]
			}
			return value
		}
	}

	return ""
}

// hasValidateRequired checks if a field has a validate:"required" tag.
func (g *OpenAPIGenerator) hasValidateRequired(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}

	tag := strings.Trim(field.Tag.Value, "`")
	return strings.Contains(tag, `validate:"required`) ||
		strings.Contains(tag, `validate:"required,`)
}
