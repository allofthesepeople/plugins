package goakit

import (
	"path"
	"regexp"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/eval"
	"goa.design/goa/expr"
	grpccodegen "goa.design/goa/grpc/codegen"
	httpcodegen "goa.design/goa/http/codegen"
)

// Register the plugin Generator functions.
func init() {
	codegen.RegisterPluginFirst("goakit", "gen", nil, Generate)
	codegen.RegisterPluginLast("goakit-goakitify", "gen", nil, Goakitify)
	codegen.RegisterPluginLast("goakit-goakitify-example", "example", nil, GoakitifyExample)
}

// Generate generates go-kit specific decoders and encoders.
func Generate(genpkg string, roots []eval.Root, files []*codegen.File) ([]*codegen.File, error) {
	for _, root := range roots {
		if r, ok := root.(*expr.RootExpr); ok {
			files = append(files, EncodeDecodeFiles(genpkg, r)...)
			files = append(files, MountFiles(r)...)
			files = append(files, ServerFiles(genpkg, r)...)
		}
	}
	return files, nil
}

// Goakitify modifies all the previously generated files by adding go-kit
// imports and replacing the following instances
// * "goa.Endpoint" with "github.com/go-kit/kit/endpoint".Endpoint
// * "log.Logger" with "github.com/go-kit/kit/log".Logger
// and adding the corresponding imports.
func Goakitify(genpkg string, roots []eval.Root, files []*codegen.File) ([]*codegen.File, error) {
	for _, f := range files {
		goakitify(f)
		for _, s := range f.SectionTemplates {
			// Don't generate the server and handler init functions in gRPC server
			// package since we generate the same in the kit server package.
			if s.Name == "server-init" {
				if _, ok := s.Data.(*grpccodegen.ServiceData); ok {
					s.Source = ""
				}
			}
			if s.Name == "handler-init" {
				if _, ok := s.Data.(*grpccodegen.EndpointData); ok {
					s.Source = ""
				}
			}
		}
	}
	return files, nil
}

// GoakitifyExample  modifies all the previously generated example files by
// adding go-kit imports.
func GoakitifyExample(genpkg string, roots []eval.Root, files []*codegen.File) ([]*codegen.File, error) {
	for _, f := range files {
		gokitifyExampleServer(genpkg, f)
	}
	return files, nil
}

// goaEndpointRegexp matches occurrences of the "goa.Endpoint" type in Go code.
var goaEndpointRegexp = regexp.MustCompile(`([^\p{L}_])goa\.Endpoint([^\p{L}_])`)

// goakitify replaces all occurrences of goa.Endpoint with
// "github.com/go-kit/kit/endpoint".Endpoint in the file section template
// sources.
func goakitify(f *codegen.File) {
	var hasEndpoint bool
	for _, s := range f.SectionTemplates {
		if !hasEndpoint {
			hasEndpoint = goaEndpointRegexp.MatchString(s.Source)
		}
		s.Source = goaEndpointRegexp.ReplaceAllString(s.Source, "${1}endpoint.Endpoint${2}")
	}
	if hasEndpoint {
		codegen.AddImport(
			f.SectionTemplates[0],
			&codegen.ImportSpec{Path: "github.com/go-kit/kit/endpoint"},
		)
	}
}

// goaLoggerRegexp matches occurrences of "logger.<function>" in Go code.
var goaLoggerRegexp = regexp.MustCompile(`logger\.\w+\((.*)\)`)

// gokitifyExampleServer imports gokit endpoint, logger, and transport
// packages in the example server implementation. It also replaces every stdlib
// logger with gokit logger.
func gokitifyExampleServer(genpkg string, file *codegen.File) {
	goakitify(file)
	var hasLogger bool
	for _, s := range file.SectionTemplates {
		if !hasLogger {
			hasLogger = strings.Contains(s.Source, "*log.Logger")
		}
		s.Source = strings.Replace(s.Source, "*log.Logger", "log.Logger", -1)
		codegen.AddImport(file.SectionTemplates[0], &codegen.ImportSpec{Path: "fmt"})
		s.Source = goaLoggerRegexp.ReplaceAllString(s.Source, "logger.Log(\"info\", fmt.Sprintf(${1}))")

		switch s.Name {
		case "server-main-logger":
			codegen.AddImport(file.SectionTemplates[0], &codegen.ImportSpec{Path: "github.com/go-kit/kit/log"})
			s.Source = gokitLoggerT
		case "server-http-logger", "server-grpc-logger":
			s.Source = ""
		case "server-http-middleware":
			s.Source = strings.Replace(s.Source, "adapter", "logger", -1)
		case "server-http-init":
			codegen.AddImport(file.SectionTemplates[0], &codegen.ImportSpec{Path: "github.com/go-kit/kit/transport/http", Name: "kithttp"})
			codegen.AddImport(file.SectionTemplates[0], &codegen.ImportSpec{Path: "github.com/go-kit/kit/endpoint"})
			data := s.Data.(map[string]interface{})
			svcs := data["Services"].([]*httpcodegen.ServiceData)
			for _, svc := range svcs {
				codegen.AddImport(file.SectionTemplates[0], &codegen.ImportSpec{
					Path: path.Join(genpkg, "http", codegen.SnakeCase(svc.Service.VarName), "kitserver"),
					Name: svc.Service.PkgName + "kitsvr",
				})
			}
			s.Source = gokitHTTPServerInitT
		case "server-grpc-init":
			data := s.Data.(map[string]interface{})
			svcs := data["Services"].([]*grpccodegen.ServiceData)
			for _, svc := range svcs {
				codegen.AddImport(file.SectionTemplates[0], &codegen.ImportSpec{
					Path: path.Join(genpkg, "grpc", codegen.SnakeCase(svc.Service.VarName), "kitserver"),
					Name: svc.Service.PkgName + "kitsvr",
				})
			}
			s.Source = strings.Replace(s.Source, "svr.New", "kitsvr.New", -1)
		case "server-grpc-register":
			s.Source = strings.Replace(s.Source, "adapter", "logger", -1)
		}
	}
	if hasLogger {
		// Replace existing stdlib logger with gokit logger in imports
		if data, ok := file.SectionTemplates[0].Data.(map[string]interface{}); ok {
			if imports, ok := data["Imports"]; ok {
				specs := imports.([]*codegen.ImportSpec)
				for _, s := range specs {
					if s.Path == "log" {
						s.Path = "github.com/go-kit/kit/log"
					}
				}
			}
		}
	}
}

const gokitLoggerT = `
  // Setup gokit logger.
  var (
    logger log.Logger
  )
  {
    logger = log.NewLogfmtLogger(os.Stderr)
    logger = log.With(logger, "ts", log.DefaultTimestampUTC)
    logger = log.With(logger, "caller", log.DefaultCaller)
  }
`

const gokitHTTPServerInitT = `
  // Wrap the endpoints with the transport specific layers. The generated
  // server packages contains code generated from the design which maps
  // the service input and output data structures to HTTP requests and
  // responses.
  var (
  {{- range .Services }}
    {{- range .Endpoints }}
      {{ .ServiceVarName }}{{ .Method.VarName }}Handler *kithttp.Server
    {{- end }}
    {{ .Service.VarName }}Server *{{.Service.PkgName}}svr.Server
  {{- end }}
  )
  {
    eh := errorHandler(logger)
    {{- if needStream .Services }}
      upgrader := &websocket.Upgrader{}
    {{- end }}
  {{- range .Services }}
    {{- if .Endpoints }}
      {{- range .Endpoints }}
        {{ .ServiceVarName }}{{ .Method.VarName }}Handler = kithttp.NewServer(
          endpoint.Endpoint({{ .ServiceVarName }}Endpoints.{{ .Method.VarName }}),
          {{- if .Payload.Ref }}
            {{ .ServicePkgName}}kitsvr.{{ .RequestDecoder }}(mux, dec),
          {{- else }}
            func(context.Context, *http.Request) (request interface{}, err error) { return nil, nil },
          {{- end }}
          {{ .ServicePkgName}}kitsvr.{{ .ResponseEncoder }}(enc),
        )
      {{- end }}
      {{ .Service.VarName }}Server = {{ .Service.PkgName }}svr.New({{ .Service.VarName }}Endpoints, mux, dec, enc, eh{{ if needStream $.Services }}, upgrader, nil{{ end }}{{ range .Endpoints }}{{ if .MultipartRequestDecoder }}, {{ $.APIPkg }}.{{ .MultipartRequestDecoder.FuncName }}{{ end }}{{ end }})
    {{-  else }}
      {{ .Service.VarName }}Server = {{ .Service.PkgName }}svr.New(nil, mux, dec, enc, eh)
    {{-  end }}
  {{- end }}
  }

  // Configure the mux.
  {{- range .Services }}{{ $service := . }}
    {{- range .Endpoints }}
  {{ .ServicePkgName}}kitsvr.{{ .MountHandler }}(mux, {{ .ServiceVarName }}{{ .Method.VarName }}Handler)
    {{- end }}
    {{- range .FileServers }}
  {{ $service.Service.PkgName}}kitsvr.{{ .MountHandler }}(mux)
    {{- end }}
  {{- end }}
`
