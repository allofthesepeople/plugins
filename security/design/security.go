package design

import (
	"fmt"
	"net/url"

	"goa.design/goa/design"
	"goa.design/goa/eval"
	httpdesign "goa.design/goa/http/design"
)

// SchemeKind is a type of security scheme.
type SchemeKind int

const (
	// OAuth2Kind identifies a "OAuth2" security scheme.
	OAuth2Kind SchemeKind = iota + 1
	// BasicAuthKind means "basic" security scheme.
	BasicAuthKind
	// APIKeyKind means "apiKey" security scheme.
	APIKeyKind
	// JWTKind means an "apiKey" security scheme, with support for
	// TokenPath and Scopes.
	JWTKind
	// NoKind means to have no security for this endpoint.
	NoKind
)

// FlowKind is a type of OAuth2 flow.
type FlowKind int

const (
	// AuthorizationCodeFlowKind identifies a OAuth2 authorization code
	// flow.
	AuthorizationCodeFlowKind FlowKind = iota + 1
	// ImplicitFlowKind identifiers a OAuth2 implicit flow.
	ImplicitFlowKind
	// PasswordFlowKind identifies a Resource Owner Password flow.
	PasswordFlowKind
	// ClientCredentialsFlowKind identifies a OAuth Client Credentials flow.
	ClientCredentialsFlowKind
)

type (
	// SecurityExpr defines a security requirement.
	SecurityExpr struct {
		// Schemes is the list of security schemes used for this
		// requirement.
		Schemes []*SchemeExpr
		// Scopes list the required scopes if any.
		Scopes []string `json:"scopes,omitempty"`
	}

	// ServiceSecurityExpr defines a security requirement that applies to
	// a service.
	ServiceSecurityExpr struct {
		*SecurityExpr
		// Service is the service that the security requirements applies
		// to.
		Service *design.ServiceExpr
	}

	// EndpointSecurityExpr defines a security requirement that applies to
	// an endpoint.
	EndpointSecurityExpr struct {
		*SecurityExpr
		// Endpoint is the endpoint that the security requirements
		// applies to.
		Method *design.MethodExpr
	}

	// SchemeExpr defines a security scheme used to authenticate against the
	// method being designed.
	SchemeExpr struct {
		// Kind is the sort of security scheme this object represents.
		Kind SchemeKind
		// SchemeName is the name of the security scheme, e.g. "googAuth",
		// "my_big_token", "jwt".
		SchemeName string
		// Description describes the security scheme e.g. "Google OAuth2"
		Description string
		// In determines the location of the API key, one of "header" or
		// "query".
		In string
		// Name refers to a header or parameter name, based on In's
		// value.
		Name string
		// Scopes lists the JWT or OAuth2 scopes.
		Scopes []*ScopeExpr
		// Flows determine the oauth2 flows supported by this scheme.
		Flows []*FlowExpr
		// Metadata is a list of key/value pairs
		Metadata design.MetadataExpr
	}

	// FlowExpr describes a specific OAuth2 flow.
	FlowExpr struct {
		// Kind is the kind of flow.
		Kind FlowKind
		// AuthorizationURL to be used for implicit or authorizationCode
		// flows.
		AuthorizationURL string
		// TokenURL to be used for password, clientCredentials or
		// authorizationCode flows.
		TokenURL string
		// RefreshURL to be used for obtaining refresh token.
		RefreshURL string
	}

	// A ScopeExpr defines a scope name and description.
	ScopeExpr struct {
		// Name of the scope.
		Name string
		// Description is the description of the scope.
		Description string
	}
)

// Requirements returns the security requirements for the endpoint ep of service
// svc.
func Requirements(svc, ep string) []*SecurityExpr {
	var sexpr []*SecurityExpr
	found := false
	for _, es := range Root.EndpointSecurity {
		if es.Method.Service.Name == svc && es.Method.Name == ep {
			// Handle special case of no security
			for _, s := range es.Schemes {
				if s.Kind == NoKind {
					sexpr = nil
					found = true
					break
				}
			}
			sexpr = append(sexpr, es.SecurityExpr)
			found = true
		}
	}
	if found {
		return sexpr
	}
	for _, ss := range Root.ServiceSecurity {
		if ss.Service.Name == svc {
			sexpr = append(sexpr, ss.SecurityExpr)
			found = true
		}
	}
	if found {
		return sexpr
	}
	return Root.APISecurity
}

// EvalName returns the generic definition name used in error messages.
func (s *SecurityExpr) EvalName() string {
	var suffix string
	if len(s.Schemes) > 0 && len(s.Schemes[0].SchemeName) > 0 {
		suffix = "scheme " + s.Schemes[0].SchemeName
	}
	return "Security" + suffix
}

// EvalName returns the generic definition name used in error messages.
func (s *SchemeExpr) EvalName() string {
	switch s.Kind {
	case OAuth2Kind:
		return "OAuth2Security"
	case BasicAuthKind:
		return "BasicAuthSecurity"
	case APIKeyKind:
		return "APIKeySecurity"
	case JWTKind:
		return "JWTSecurity"
	default:
		return "[unknown]"
	}
}

// Validate ensures that TokenURL and AuthorizationURL are valid URLs.
func (s *SchemeExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	payloads := make(map[string]*design.AttributeExpr)
	for _, svc := range design.Root.Services {
		for _, m := range svc.Methods {
			found := false
			for _, req := range Requirements(svc.Name, m.Name) {
				for _, scheme := range req.Schemes {
					if scheme == s {
						loc := fmt.Sprintf("method %q of service %q", m.Name, svc.Name)
						payloads[loc] = m.Payload
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}
	for loc, payload := range payloads {
		switch s.Kind {
		case BasicAuthKind:
			if !hasTaggedField(payload, "security:username") {
				verr.Add(s, "payload of %s does not define a username attribute, use Username to define one.", loc)
			}
			if !hasTaggedField(payload, "security:password") {
				verr.Add(s, "payload of %s does not define a password attribute, use Password to define one.", loc)
			}
		case APIKeyKind:
			if !hasTaggedField(payload, "security:apikey:"+s.SchemeName) {
				verr.Add(s, "payload of %s does not define an API key attribute, use APIKey to define one.", loc)
			}
		case JWTKind:
			if !hasTaggedField(payload, "security:token") {
				verr.Add(s, "payload of %s does not define a JWT attribute, use Token to define one.", loc)
			}
		case OAuth2Kind:
			if !hasTaggedField(payload, "security:accesstoken") {
				verr.Add(s, "payload of %s does not define a OAuth2 access token attribute, use AccessToken to define one.", loc)
			}
		default:
			panic(fmt.Sprintf("unknown kind %#v", s.Kind)) // bug
		}
	}
	for _, f := range s.Flows {
		if err := f.Validate(); err != nil {
			verr.Merge(err)
		}
	}
	return verr
}

// Finalize makes the TokenURL and AuthorizationURL complete if needed.
func (s *SchemeExpr) Finalize() {
	for _, f := range s.Flows {
		f.Finalize()
	}
	if s.Kind == OAuth2Kind || s.Kind == JWTKind {
		if s.Name == "" {
			s.Name = "Authorization"
		}
	}
}

// EvalName returns the name of the expression used in error messages.
func (s *FlowExpr) EvalName() string {
	if s.TokenURL != "" {
		return fmt.Sprintf("flow with token URL %q", s.TokenURL)
	}
	return fmt.Sprintf("flow with refresh URL %q", s.RefreshURL)
}

// Validate ensures that TokenURL and AuthorizationURL are valid URLs.
func (s *FlowExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	_, err := url.Parse(s.TokenURL)
	if err != nil {
		verr.Add(s, "invalid token URL %q: %s", s.TokenURL, err)
	}
	_, err = url.Parse(s.AuthorizationURL)
	if err != nil {
		verr.Add(s, "invalid authorization URL %q: %s", s.AuthorizationURL, err)
	}
	_, err = url.Parse(s.RefreshURL)
	if err != nil {
		verr.Add(s, "invalid refresh URL %q: %s", s.RefreshURL, err)
	}
	return verr
}

// Finalize makes the TokenURL and AuthorizationURL complete if needed.
func (s *FlowExpr) Finalize() {
	tu, _ := url.Parse(s.TokenURL)         // validated in Validate
	au, _ := url.Parse(s.AuthorizationURL) // validated in Validate
	ru, _ := url.Parse(s.RefreshURL)       // validated in Validate
	tokenOK := s.TokenURL == "" || tu.IsAbs()
	authOK := s.AuthorizationURL == "" || au.IsAbs()
	refreshOK := s.RefreshURL == "" || ru.IsAbs()
	if tokenOK && authOK && refreshOK {
		return
	}
	var (
		scheme string
		host   string
	)
	if len(httpdesign.Root.Design.API.Servers) > 0 {
		u, _ := url.Parse(httpdesign.Root.Design.API.Servers[0].URL)
		scheme = u.Scheme
		host = u.Host
	}
	if !tokenOK {
		tu.Scheme = scheme
		tu.Host = host
		s.TokenURL = tu.String()
	}
	if !authOK {
		au.Scheme = scheme
		au.Host = host
		s.AuthorizationURL = au.String()
	}
	if !refreshOK {
		ru.Scheme = scheme
		ru.Host = host
		s.RefreshURL = ru.String()
	}
}

// hasTaggedField returns true if the given attribute is an object that has an
// attribute with the given tag.
func hasTaggedField(att *design.AttributeExpr, tag string) bool {
	obj := design.AsObject(att.Type)
	if obj == nil {
		return false
	}
	for _, at := range *obj {
		if _, ok := at.Attribute.Metadata[tag]; ok {
			return true
		}
	}
	return false
}