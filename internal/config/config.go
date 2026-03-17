package config

type Config struct {
	ModulePath  string            `yaml:"module_path"`
	ProjectRoot string            `yaml:"project_root"`
	OutputDir   string            `yaml:"output_dir"`
	Info        InfoConfig        `yaml:"info"`
	Servers     []ServerConfig    `yaml:"servers"`
	Routers     []RouterConfig    `yaml:"routers"`
	Components  *ComponentsConfig `yaml:"components,omitempty"`
}

type InfoConfig struct {
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Version     string         `yaml:"version"`
	Contact     *ContactConfig `yaml:"contact,omitempty"`
	License     *LicenseConfig `yaml:"license,omitempty"`
}

type ContactConfig struct {
	Name  string `yaml:"name"`
	URL   string `yaml:"url"`
	Email string `yaml:"email"`
}

type LicenseConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type ServerConfig struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
}

type RouterConfig struct {
	Source string `yaml:"source"`

	Output string `yaml:"output"`

	Info *InfoConfig `yaml:"info,omitempty"`

	BasePath string `yaml:"base_path"`

	Tags []TagConfig `yaml:"tags,omitempty"`

	SecuritySchemes map[string]SecuritySchemeConfig `yaml:"security_schemes,omitempty"`

	Include []string `yaml:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`

	Components *ComponentsConfig `yaml:"components,omitempty"`
}

type TagConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type ComponentsConfig struct {
	Schemas         map[string]interface{}          `yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecuritySchemeConfig `yaml:"security_schemes,omitempty"`
}

type SecuritySchemeConfig struct {
	Type         string `yaml:"type"`
	Scheme       string `yaml:"scheme,omitempty"`
	BearerFormat string `yaml:"bearer_format,omitempty"`
	In           string `yaml:"in,omitempty"`
	Name         string `yaml:"name,omitempty"`
	Description  string `yaml:"description,omitempty"`
}
