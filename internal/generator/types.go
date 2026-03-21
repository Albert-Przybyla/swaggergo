package generator

type OpenAPISpec struct {
	OpenAPI    string                `yaml:"openapi"`
	Info       InfoObject            `yaml:"info"`
	Servers    []ServerObject        `yaml:"servers,omitempty"`
	Paths      map[string]*PathItem  `yaml:"paths"`
	Components *Components           `yaml:"components,omitempty"`
	Tags       []TagObject           `yaml:"tags,omitempty"`
	Security   []SecurityRequirement `yaml:"security,omitempty"`
}

type InfoObject struct {
	Title       string         `yaml:"title"`
	Description string         `yaml:"description,omitempty"`
	Version     string         `yaml:"version"`
	Contact     *ContactObject `yaml:"contact,omitempty"`
	License     *LicenseObject `yaml:"license,omitempty"`
}

type ContactObject struct {
	Name  string `yaml:"name,omitempty"`
	URL   string `yaml:"url,omitempty"`
	Email string `yaml:"email,omitempty"`
}

type LicenseObject struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

type ServerObject struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

type TagObject struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type PathItem struct {
	Ref         string `yaml:"$ref,omitempty"`
	Summary     string `yaml:"summary,omitempty"`
	Description string `yaml:"description,omitempty"`

	Get     *Operation `yaml:"get,omitempty"`
	Post    *Operation `yaml:"post,omitempty"`
	Put     *Operation `yaml:"put,omitempty"`
	Delete  *Operation `yaml:"delete,omitempty"`
	Patch   *Operation `yaml:"patch,omitempty"`
	Options *Operation `yaml:"options,omitempty"`
	Head    *Operation `yaml:"head,omitempty"`
}

type Operation struct {
	OperationID string                 `yaml:"operationId,omitempty"`
	Summary     string                 `yaml:"summary,omitempty"`
	Description string                 `yaml:"description,omitempty"`
	Tags        []string               `yaml:"tags,omitempty"`
	Parameters  []Parameter            `yaml:"parameters,omitempty"`
	RequestBody *RequestBody           `yaml:"requestBody,omitempty"`
	Responses   map[string]Response    `yaml:"responses"`
	Security    *[]SecurityRequirement `yaml:"security,omitempty"`
	Deprecated  bool                   `yaml:"deprecated,omitempty"`
	Callbacks   map[string]interface{} `yaml:"callbacks,omitempty"`
}

type Parameter struct {
	Name        string      `yaml:"name"`
	In          string      `yaml:"in"`
	Description string      `yaml:"description,omitempty"`
	Required    bool        `yaml:"required,omitempty"`
	Schema      *Schema     `yaml:"schema,omitempty"`
	Example     interface{} `yaml:"example,omitempty"`
	Style       string      `yaml:"style,omitempty"`
	Explode     *bool       `yaml:"explode,omitempty"`
}

type RequestBody struct {
	Description string               `yaml:"description,omitempty"`
	Required    bool                 `yaml:"required,omitempty"`
	Content     map[string]MediaType `yaml:"content"`
}

type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty"`
	Headers     map[string]Header    `yaml:"headers,omitempty"`
	Links       map[string]Link      `yaml:"links,omitempty"`
}

type MediaType struct {
	Schema   *Schema            `yaml:"schema,omitempty"`
	Example  interface{}        `yaml:"example,omitempty"`
	Examples map[string]Example `yaml:"examples,omitempty"`
}

type Example struct {
	Summary     string      `yaml:"summary,omitempty"`
	Description string      `yaml:"description,omitempty"`
	Value       interface{} `yaml:"value,omitempty"`
}

type Header struct {
	Description string  `yaml:"description,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty"`
}

type Schema struct {
	Ref         string `yaml:"$ref,omitempty"`
	Type        string `yaml:"type,omitempty"`
	Format      string `yaml:"format,omitempty"`
	Description string `yaml:"description,omitempty"`

	Properties map[string]*Schema `yaml:"properties,omitempty"`
	Items      *Schema            `yaml:"items,omitempty"`

	Required []string      `yaml:"required,omitempty"`
	Enum     []interface{} `yaml:"enum,omitempty"`
	Example  interface{}   `yaml:"example,omitempty"`
	Default  interface{}   `yaml:"default,omitempty"`

	Nullable   bool `yaml:"nullable,omitempty"`
	ReadOnly   bool `yaml:"readOnly,omitempty"`
	WriteOnly  bool `yaml:"writeOnly,omitempty"`
	Deprecated bool `yaml:"deprecated,omitempty"`

	AllOf []*Schema `yaml:"allOf,omitempty"`
	OneOf []*Schema `yaml:"oneOf,omitempty"`
	AnyOf []*Schema `yaml:"anyOf,omitempty"`

	AdditionalProperties interface{} `yaml:"additionalProperties,omitempty"`

	Minimum      *float64 `yaml:"minimum,omitempty"`
	Maximum      *float64 `yaml:"maximum,omitempty"`
	ExclusiveMin bool     `yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMax bool     `yaml:"exclusiveMaximum,omitempty"`

	MinLength *int   `yaml:"minLength,omitempty"`
	MaxLength *int   `yaml:"maxLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty"`

	MinItems *int `yaml:"minItems,omitempty"`
	MaxItems *int `yaml:"maxItems,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema        `yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `yaml:"securitySchemes,omitempty"`
	Parameters      map[string]Parameter      `yaml:"parameters,omitempty"`
	Responses       map[string]Response       `yaml:"responses,omitempty"`
	RequestBodies   map[string]RequestBody    `yaml:"requestBodies,omitempty"`
	Headers         map[string]Header         `yaml:"headers,omitempty"`
}

type SecurityScheme struct {
	Type         string `yaml:"type"`
	Scheme       string `yaml:"scheme,omitempty"`
	BearerFormat string `yaml:"bearerFormat,omitempty"`
	In           string `yaml:"in,omitempty"`
	Name         string `yaml:"name,omitempty"`
	Description  string `yaml:"description,omitempty"`

	Flows *OAuthFlows `yaml:"flows,omitempty"`
}

type OAuthFlows struct {
	Implicit          *OAuthFlow `yaml:"implicit,omitempty"`
	Password          *OAuthFlow `yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `yaml:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string            `yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `yaml:"scopes,omitempty"`
}

type Link struct {
	OperationID string                 `yaml:"operationId,omitempty"`
	Parameters  map[string]interface{} `yaml:"parameters,omitempty"`
}

type SecurityRequirement map[string][]string
