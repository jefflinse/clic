// Package openapi compiles an OpenAPI 3.x document into a clic spec.
package openapi

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/oas"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/provider/rest"
	"github.com/jefflinse/clic/spec"
)

// Compile parses an OpenAPI 3.x document and compiles it into a clic spec.
func Compile(data []byte) (*spec.App, error) {
	probe := map[string]any{}
	if err := ioutil.Unmarshal(data, &probe); err == nil {
		if _, ok := probe["swagger"]; ok {
			return nil, fmt.Errorf("Swagger/OpenAPI 2.0 is not supported; please use OpenAPI 3.x")
		}
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	c := newCompiler(doc)

	app := &spec.App{
		Name:        appName(doc),
		Description: appDescription(doc),
		Server:      serverURL(doc),
		Auth:        authScheme(doc),
	}

	root := &group{children: map[string]*group{}}
	for _, path := range c.sortedPaths {
		item := doc.Paths.Value(path)
		for _, method := range sortedMethods(item.Operations()) {
			groupPath, verb := c.commandPath(path, method)
			cmd, err := c.command(verb, app.Server, path, method, item)
			if err != nil {
				return nil, err
			}
			root.insert(groupPath, cmd)
		}
	}

	app.Commands = root.commands()
	return app, nil
}

// compiler holds precomputed views of the document used during mapping.
type compiler struct {
	doc         *openapi3.T
	sortedPaths []string
	methods     map[string]map[string]bool // path -> set of methods
}

func newCompiler(doc *openapi3.T) *compiler {
	c := &compiler{doc: doc, methods: map[string]map[string]bool{}}
	for path, item := range doc.Paths.Map() {
		c.sortedPaths = append(c.sortedPaths, path)
		set := map[string]bool{}
		for method := range item.Operations() {
			set[strings.ToUpper(method)] = true
		}
		c.methods[path] = set
	}
	sort.Strings(c.sortedPaths)
	return c
}

// commandPath maps an operation to its command group path and verb.
func (c *compiler) commandPath(path, method string) (groupPath []string, verb string) {
	segments := splitPath(path)
	nonParam := nonParamSegments(segments)

	if len(segments) == 0 {
		// root path "/" -> a top-level command named by the method verb
		return nil, collectionVerb(method)
	}

	if isParam(segments[len(segments)-1]) {
		// item operation, e.g. GET /users/{id} -> users get <id>
		return nonParam, c.itemVerb(path, method)
	}

	// trailing non-param segment that follows a path parameter and is a single,
	// childless operation is treated as an action verb, e.g. POST /users/{id}/activate
	tail := segments[len(segments)-1]
	precededByParam := len(segments) >= 2 && isParam(segments[len(segments)-2])
	if precededByParam && c.isAction(path) {
		return nonParam[:len(nonParam)-1], tail
	}

	// collection operation, e.g. GET /users -> users list
	return nonParam, collectionVerb(method)
}

// itemVerb returns the verb for an operation on a single item.
func (c *compiler) itemVerb(path, method string) string {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return "get"
	case http.MethodDelete:
		return "delete"
	case http.MethodPatch:
		return "update"
	case http.MethodPut:
		if c.methods[path][http.MethodPatch] {
			return "replace"
		}
		return "update"
	case http.MethodPost:
		return "create"
	default:
		return strings.ToLower(method)
	}
}

// collectionVerb returns the verb for an operation on a collection.
func collectionVerb(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return "list"
	case http.MethodPost:
		return "create"
	case http.MethodPut:
		return "replace"
	case http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// isAction reports whether a path is a leaf action (single operation, no child
// paths) rather than a sub-resource collection.
func (c *compiler) isAction(path string) bool {
	if len(c.methods[path]) != 1 {
		return false
	}
	prefix := path + "/"
	for _, other := range c.sortedPaths {
		if strings.HasPrefix(other, prefix) {
			return false
		}
	}
	return true
}

// command builds a leaf command (with a rest provider) for an operation.
func (c *compiler) command(verb, base, path, method string, item *openapi3.PathItem) (*spec.Command, error) {
	op := item.Operations()[strings.ToUpper(method)]

	restSpec := &rest.Spec{
		BaseURL:   base,
		Endpoint:  path,
		Method:    strings.ToUpper(method),
		RawBody:   op.RequestBody != nil,
		Body:      BodyFields(requestBodySchema(op.RequestBody)),
		Responses: oas.Extract(op),
	}

	// path-item parameters apply to every operation; operation parameters override
	for _, ref := range append(append(openapi3.Parameters{}, item.Parameters...), op.Parameters...) {
		p := ref.Value
		if p == nil {
			continue
		}

		param := &provider.Parameter{
			Name:        p.Name,
			Description: p.Description,
			Type:        schemaType(p.Schema),
			Required:    p.Required,
		}

		switch p.In {
		case openapi3.ParameterInPath:
			param.Required = true
			restSpec.PathParams = append(restSpec.PathParams, param)
		case openapi3.ParameterInQuery:
			restSpec.QueryParams = append(restSpec.QueryParams, param)
		case openapi3.ParameterInHeader:
			restSpec.HeaderParams = append(restSpec.HeaderParams, param)
		}
	}

	return &spec.Command{
		Name:        verb,
		Description: operationDescription(op, verb, path),
		Provider:    restSpec,
	}, nil
}

// group is a node in the command tree built from path segments.
type group struct {
	children map[string]*group
	cmds     []*spec.Command
}

func (g *group) insert(groupPath []string, cmd *spec.Command) {
	if len(groupPath) == 0 {
		g.cmds = append(g.cmds, cmd)
		return
	}

	name := groupPath[0]
	child, ok := g.children[name]
	if !ok {
		child = &group{children: map[string]*group{}}
		g.children[name] = child
	}
	child.insert(groupPath[1:], cmd)
}

// commands converts the group tree into a sorted slice of clic commands.
func (g *group) commands() []*spec.Command {
	cmds := append([]*spec.Command{}, g.cmds...)

	names := make([]string, 0, len(g.children))
	for name := range g.children {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		child := g.children[name]
		cmds = append(cmds, &spec.Command{
			Name:        name,
			Description: fmt.Sprintf("%s commands", name),
			Subcommands: child.commands(),
		})
	}

	sort.SliceStable(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
	uniquifyNames(cmds)
	return cmds
}

// uniquifyNames renames any commands that share a name within a group so the
// resulting command tree has no collisions. Colliding rest commands prefer a
// method suffix (e.g. create-post); anything still ambiguous gets a numeric one.
func uniquifyNames(cmds []*spec.Command) {
	used := map[string]bool{}
	for _, cmd := range cmds {
		if !used[cmd.Name] {
			used[cmd.Name] = true
			continue
		}

		base := cmd.Name
		if r, ok := cmd.Provider.(*rest.Spec); ok && r != nil {
			if candidate := base + "-" + strings.ToLower(r.Method); !used[candidate] {
				cmd.Name = candidate
				used[candidate] = true
				continue
			}
		}

		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s-%d", base, i)
			if !used[candidate] {
				cmd.Name = candidate
				used[candidate] = true
				break
			}
		}
	}
}

func appName(doc *openapi3.T) string {
	if doc.Info != nil {
		if s := slug(doc.Info.Title); s != "" {
			return s
		}
	}
	return "app"
}

func appDescription(doc *openapi3.T) string {
	if doc.Info != nil {
		if doc.Info.Description != "" {
			return firstLine(doc.Info.Description)
		}
		if doc.Info.Title != "" {
			return doc.Info.Title
		}
	}
	return "generated from an OpenAPI document"
}

func operationDescription(op *openapi3.Operation, verb, path string) string {
	if op.Summary != "" {
		return op.Summary
	}
	if op.Description != "" {
		return firstLine(op.Description)
	}
	return fmt.Sprintf("%s %s", verb, path)
}

func serverURL(doc *openapi3.T) string {
	if len(doc.Servers) == 0 {
		return ""
	}

	server := doc.Servers[0]
	url := server.URL
	for name, variable := range server.Variables {
		if variable != nil {
			url = strings.ReplaceAll(url, "{"+name+"}", variable.Default)
		}
	}
	return strings.TrimRight(url, "/")
}

func authScheme(doc *openapi3.T) *provider.AuthScheme {
	if doc.Components == nil || len(doc.Components.SecuritySchemes) == 0 {
		return nil
	}

	names := make([]string, 0, len(doc.Components.SecuritySchemes))
	for name := range doc.Components.SecuritySchemes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ref := doc.Components.SecuritySchemes[name]
		if ref == nil || ref.Value == nil {
			continue
		}
		scheme := ref.Value

		switch strings.ToLower(scheme.Type) {
		case "http":
			switch strings.ToLower(scheme.Scheme) {
			case "bearer":
				return &provider.AuthScheme{Type: provider.AuthBearer}
			case "basic":
				return &provider.AuthScheme{Type: provider.AuthBasic}
			}
		case "apikey":
			return &provider.AuthScheme{Type: provider.AuthAPIKey, In: scheme.In, Name: scheme.Name}
		case "oauth2":
			if s := oauthScheme(scheme.Flows); s != nil {
				return s
			}
		}
	}

	return nil
}

// oauthScheme maps an OpenAPI oauth2 securityScheme's flows onto a clic auth
// scheme. It prefers the non-interactive client-credentials flow, falling back
// to authorization-code; a --oauth-flow override is applied later at token
// resolution. Returns nil when neither supported flow is declared.
func oauthScheme(flows *openapi3.OAuthFlows) *provider.AuthScheme {
	if flows == nil {
		return nil
	}
	if f := flows.ClientCredentials; f != nil {
		return &provider.AuthScheme{
			Type:     provider.AuthOAuth2,
			Flow:     provider.FlowClientCredentials,
			TokenURL: f.TokenURL,
			Scopes:   scopeNames(f.Scopes),
		}
	}
	if f := flows.AuthorizationCode; f != nil {
		return &provider.AuthScheme{
			Type:     provider.AuthOAuth2,
			Flow:     provider.FlowAuthorizationCode,
			AuthURL:  f.AuthorizationURL,
			TokenURL: f.TokenURL,
			Scopes:   scopeNames(f.Scopes),
		}
	}
	return nil
}

// scopeNames returns the scope identifiers from an OAuth flow's scope map, sorted
// for stable output.
func scopeNames(scopes map[string]string) []string {
	if len(scopes) == 0 {
		return nil
	}
	names := make([]string, 0, len(scopes))
	for name := range scopes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func schemaType(ref *openapi3.SchemaRef) string {
	if ref == nil || ref.Value == nil || ref.Value.Type == nil {
		return provider.StringParamType
	}

	t := ref.Value.Type
	switch {
	case t.Is("integer"):
		return provider.IntParamType
	case t.Is("number"):
		return provider.NumberParamType
	case t.Is("boolean"):
		return provider.BoolParamType
	default:
		return provider.StringParamType
	}
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}

func splitPath(path string) []string {
	segments := []string{}
	for s := range strings.SplitSeq(path, "/") {
		if s != "" {
			segments = append(segments, s)
		}
	}
	return segments
}

func nonParamSegments(segments []string) []string {
	out := []string{}
	for _, s := range segments {
		if !isParam(s) {
			out = append(out, s)
		}
	}
	return out
}

func isParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func sortedMethods(ops map[string]*openapi3.Operation) []string {
	methods := make([]string, 0, len(ops))
	for method := range ops {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}
