package plugin

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	av_opts "github.com/infobloxopen/protoc-gen-atlas-validate/options"
)

const (
	// PluginName is name of the plugin specified for protoc
	PluginName = "atlas-validate"

)

type Plugin struct {

	*generator.Generator
	*pluginImports

	file    *generator.FileDescriptor
	methods map[string][]*methodDescriptor
	imports map[string]*importPkg
	fcount  int

	annotatorOnce sync.Once
}


func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Init(g *generator.Generator) {
	p.Generator = g

	p.methods = make(map[string][]*methodDescriptor)
	for _, f := range p.Generator.Request.ProtoFile {
		for _, fg := range p.Generator.Request.FileToGenerate {
			if f.GetName() == fg {
				p.methods[f.GetName()] = p.gatherMethods(f)
				p.fcount++
			}
		}
	}

}

func (p *Plugin) GenerateImports(file *generator.FileDescriptor) {
	if p.pluginImports != nil {
		p.pluginImports.GenerateImports(file)
	}
}

func (p *Plugin) Generate(file *generator.FileDescriptor) {
	if _, ok := p.methods[file.GetName()]; !ok {
		return
	}

	p.fcount--

	p.file = file

	p.initPluginImports(p.Generator)

	p.renderValidatorMethods()
	p.renderValidatorObjectMethods()

	if p.fcount == 0 || strings.HasSuffix(file.GetName(), file.GetPackage()+".proto") {
		p.annotatorOnce.Do(func() {
			p.renderMethodDescriptors()
			p.renderAnnotator()
		})
	}
}

// getAllowUnknown function picks up correct allowUnknown option from file/service/method
// hierarchy.
func (p *Plugin) getAllowUnknown(file proto.Message, svc proto.Message, method proto.Message) bool {
	var gavOpt *av_opts.AtlasValidateFileOption
	if aExt, err := proto.GetExtension(file, av_opts.E_File); err == nil && aExt != nil {
		gavOpt = aExt.(*av_opts.AtlasValidateFileOption)
	}

	var savOpt *av_opts.AtlasValidateServiceOption
	if aExt, err := proto.GetExtension(svc, av_opts.E_Service); err == nil && aExt != nil {
		savOpt = aExt.(*av_opts.AtlasValidateServiceOption)
	}

	var mavOpt *av_opts.AtlasValidateMethodOption
	if aExt, err := proto.GetExtension(method, av_opts.E_Method); err == nil && aExt != nil {
		mavOpt = aExt.(*av_opts.AtlasValidateMethodOption)
	}

	if mavOpt != nil {
		return mavOpt.GetAllowUnknownFields()
	} else if savOpt != nil {
		return savOpt.GetAllowUnknownFields()
	}

	return gavOpt.GetAllowUnknownFields()
}

type methodDescriptor struct {
	svc                  string
	method               string
	idx                  int
	httpBody, httpMethod string
	gwPattern            string
	allowUnknown         bool
	inputType            string
}

// gatherMethods function walks through services and methods and extracts
// google.api.http options and renders different handlers for HTTP request/pattern pair.
func (p *Plugin) gatherMethods(f *descriptor.FileDescriptorProto) []*methodDescriptor {

	var methods []*methodDescriptor

	for _, svc := range f.GetService() {
		for _, method := range svc.GetMethod() {
			for i, opt := range extractHTTPOpts(method) {
				methods = append(methods, &methodDescriptor{
					svc:          svc.GetName(),
					method:       method.GetName(),
					idx:          i,
					httpBody:     opt.body,
					httpMethod:   opt.method,
					gwPattern:    fmt.Sprintf("%s_%s_%d", svc.GetName(), method.GetName(), i),
					inputType:    method.GetInputType(),
					allowUnknown: p.getAllowUnknown(f.Options, svc.Options, method.Options),
				})
			}
		}
	}

	return methods
}

// renderMethodDescriptors renders array of structs that are used to trigger validation
// function on correct HTTP request according to HTTP method and grpc-gateway/runtime.Pattern.
func (p *Plugin) renderMethodDescriptors() {

	var (
		jsonPkg      = p.NewImport(jsonPkgPath)
		ctxPkg       = p.NewImport(ctxPkgPath)
		gwruntimePkg = p.NewImport(gwruntimePkgPath)
	)

	p.P(`var validate_Patterns = []struct{`)
	p.P(`pattern `, gwruntimePkg.Use(), `.Pattern`)
	p.P(`httpMethod string`)
	p.P(`validator func(`, ctxPkg.Use(), `.Context, `, jsonPkg.Use(), `.RawMessage) error`)
	p.P(`// Included for introspection purpose.`)
	p.P(`allowUnknown bool`)
	p.P(`} {`)
	for f, methods := range p.methods {
		p.P(`// patterns for file `, f)
		for _, m := range methods {
			p.P(`{`)
			// NOTE: pattern reiles on code generated by protoc-gen-grpc-gateway.
			p.P(`pattern: `, "pattern_"+m.gwPattern, `,`)
			p.P(`httpMethod: "`, m.httpMethod, `",`)
			p.P(`validator: `, "validate_"+m.gwPattern, `,`)
			p.P(`allowUnknown: `, m.allowUnknown, `,`)
			p.P(`},`)
		}
		p.P()
	}
	p.P(`}`)
	p.P()
}

// renderValidatorMethods function generates entrypoints for validator one per each
// HTTP request (and HTTP request additional_bindings).
func (p *Plugin) renderValidatorMethods() {

	var (
		fmtPkg  = p.NewImport(fmtPkgPath)
		jsonPkg = p.NewImport(jsonPkgPath)
		ctxPkg  = p.NewImport(ctxPkgPath)
	)

	for _, m := range p.methods[p.file.GetName()] {
		p.P(`// validate_`, m.gwPattern, ` is an entrypoint for validating "`, m.httpMethod, `" HTTP request `)
		p.P(`// that match *.pb.gw.go/pattern_`, m.gwPattern, `.`)
		p.P(`func validate_`, m.gwPattern, `(ctx `, ctxPkg.Use(), `.Context, r `, jsonPkg.Use(), `.RawMessage) (err error) {`)

		o := p.objectNamed(m.inputType)
		t := p.TypeName(o)

		switch m.httpBody {
		case "":

			p.P(`if len(r) != 0 {`)
			p.P(`return `, fmtPkg.Use(), `.Errorf("body is not allowed")`)
			p.P(`}`)
			p.P(`return nil`)

		case "*":

			if p.isLocal(o) {
				p.P(`return validate_Object_`, t, `(ctx, r, "")`)
			} else {
				p.P(`if validator, ok := `, p.generateAtlasValidateJSONInterfaceSignature(t), `; ok {`)
				p.P(`return validator.AtlasValidateJSON(ctx, r, "")`)
				p.P(`}`)
				p.P(`return nil`)
			}

		default:

			fo := p.objectFieldNamed(o, t, m.httpBody)
			ft := p.TypeName(fo)

			if p.isLocal(fo) {
				p.P(`return validate_Object_`, ft, `(ctx, r, "")`)
			} else {
				p.P(`if validator, ok := `, p.generateAtlasValidateJSONInterfaceSignature(ft), `; ok {`)
				p.P(`return validator.AtlasValidateJSON(ctx, r, "")`)
				p.P(`}`)
				p.P(`return nil`)
			}
		}
		p.P(`}`)
		p.P()
	}
}

func (p *Plugin) renderValidatorObjectMethods() {

	for _, o := range p.file.GetMessageType() {

		ptype := "." + p.file.GetPackage() + "." + o.GetName()
		otype := p.TypeName(p.objectNamed(ptype))

		p.renderValidatorObjectMethod(o, otype)
		p.generateValidateRequired(o, otype)

		for _, no := range o.GetNestedType() {

			if no.GetOptions().GetMapEntry() {
				continue
			}

			notype := p.TypeName(p.objectNamed(ptype + "." + no.GetName()))

			p.renderValidatorObjectMethod(no, notype)
			p.generateValidateRequired(no, notype)
		}
	}
}

func (p *Plugin) renderValidatorObjectMethod(o *descriptor.DescriptorProto, t string) {

	var (
		jsonPkg    = p.NewImport(jsonPkgPath)
		fmtPkg     = p.NewImport(fmtPkgPath)
		ctxPkg     = p.NewImport(ctxPkgPath)
		runtimePkg = p.NewImport(runtimePkgPath)
	)

	p.P(`// validate_Object_`, t, ` function validates a JSON for a given object.`)
	p.P(`func validate_Object_`, t, `(ctx `, ctxPkg.Use(), `.Context, r `, jsonPkg.Use(), `.RawMessage, path string) (err error) {`)
	p.P(`if hook, ok := `, p.generateAtlasJSONValidateInterfaceSignature(t), `; ok {`)
	p.P(`if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {`)
	p.P(`return err`)
	p.P(`}`)
	p.P(`}`)
	p.P(`var v map[string]`, jsonPkg.Use(), `.RawMessage`)
	p.P(`if err = `, jsonPkg.Use(), `.Unmarshal(r, &v); err != nil {`)
	p.P(`return `, fmtPkg.Use(), `.Errorf("invalid value for %q: expected object.", path)`)
	p.P(`}`)
	p.P(`allowUnknown := `, runtimePkg.Use(), `.AllowUnknownFromContext(ctx)`)
	p.P()
	p.P(`if err = validate_required_Object_`, t, `(ctx, v, path); err != nil {`)
	p.P(`return err`)
	p.P(`}`)
	p.P()
	p.P(`for k, _ := range v {`)

	p.P(`switch k {`)
	for _, f := range o.GetField() {

		p.P(`case "`, f.GetName(), `":`)

		if p.IsMap(f) {
			continue
		}

		if fExt, err := proto.GetExtension(f.Options, av_opts.E_Field); err == nil && fExt != nil {
			favOpt := fExt.(*av_opts.AtlasValidateFieldOption)
			methods := p.GetDeniedMethods(favOpt.GetDeny())
			if len(methods) != 0 {
				cond := strings.Join(methods, `" || method == "`)
				p.P(`method := `, runtimePkg.Use(), `.HTTPMethodFromContext(ctx)`)
				p.P(`if method == "`, cond, `" {`)
				p.P(`return `, fmtPkg.Use(), `.Errorf("field %q is unsupported for %q operation.", k, method)`)
				p.P("}")
			}
		}

		if f.IsMessage() && f.IsRepeated() {

			fo := p.objectNamed(f.GetTypeName())
			ft := p.TypeName(fo)

			p.P(`if v[k] == nil {`)
			p.P(`continue`)
			p.P(`}`)
			p.P(`var vArr []`, jsonPkg.Use(), `.RawMessage`)
			p.P(`vArrPath := `, runtimePkg.Use(), `.JoinPath(path, k)`)
			p.P(`if err = `, jsonPkg.Use(), `.Unmarshal(v[k], &vArr); err != nil {`)
			p.P(`return `, fmtPkg.Use(), `.Errorf("invalid value for %q: expected array.", vArrPath)`)
			p.P(`}`)
			if !p.isLocal(fo) {
				p.P(`validator, ok := `, p.generateAtlasValidateJSONInterfaceSignature(ft))
				p.P(`if !ok {`)
				p.P(`continue`)
				p.P(`}`)
			}
			p.P(`for i, vv := range vArr {`)
			p.P(`vvPath := `, fmtPkg.Use(), `.Sprintf("%s.[%d]", vArrPath, i)`)
			if !p.isLocal(fo) {
				p.P(`if err = validator.AtlasValidateJSON(ctx, vv, vvPath); err != nil {`)
				p.P(`return err`)
				p.P(`}`)
			} else {
				p.P(`if err = validate_Object_`, ft, `(ctx, vv, vvPath); err != nil {`)
				p.P(`return err`)
				p.P(`}`)
			}
			p.P(`}`)

		} else if f.IsMessage() {

			fo := p.objectNamed(f.GetTypeName())
			ft := p.TypeName(fo)

			p.P(`if v[k] == nil {`)
			p.P(`continue`)
			p.P(`}`)
			p.P(`vv := v[k]`)
			p.P(`vvPath := `, runtimePkg.Use(), `.JoinPath(path, k)`)
			if p.isLocal(fo) {
				p.P(`if err = validate_Object_`, ft, `(ctx, vv, vvPath); err != nil {`)
				p.P(`return err`)
				p.P(`}`)
			} else {
				p.P(`validator, ok := `, p.generateAtlasValidateJSONInterfaceSignature(ft))
				p.P(`if !ok {`)
				p.P(`continue`)
				p.P(`}`)
				p.P(`if err = validator.AtlasValidateJSON(ctx, vv, vvPath); err != nil {`)
				p.P(`return err`)
				p.P(`}`)
			}
		}
	}

	p.P(`default:`)
	p.P(`if !allowUnknown {`)
	p.P(`return `, fmtPkg.Use(), `.Errorf("unknown field %q.", `, runtimePkg.Use(), `.JoinPath(path, k))`)
	p.P(`}`)
	p.P(`}`)
	p.P(`}`)
	p.P(`return nil`)
	p.P(`}`)
	p.P()

	p.P(`// AtlasValidateJSON function validates a JSON for object `, t, `.`)
	p.P(`func (_ *`, t, `) AtlasValidateJSON(ctx `, ctxPkg.Use(), `.Context, r `, jsonPkg.Use(), `.RawMessage, path string) (err error) {`)
	p.P(`if hook, ok := `, p.generateAtlasJSONValidateInterfaceSignature(t), `; ok {`)
	p.P(`if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {`)
	p.P(`return err`)
	p.P(`}`)
	p.P(`}`)
	p.P(`return validate_Object_`, t, `(ctx, r, path)`)
	p.P(`}`)
	p.P()
}

func (p *Plugin) generateAtlasValidateJSONInterfaceSignature(t string) string {

	var (
		jsonPkg = p.NewImport(jsonPkgPath)
		ctxPkg  = p.NewImport(ctxPkgPath)
	)

	return fmt.Sprintf(`interface{}(&%s{}).(interface{ AtlasValidateJSON(%s.Context, %s.RawMessage, string) error })`, t, ctxPkg.Use(), jsonPkg.Use())
}

func (p *Plugin) generateAtlasJSONValidateInterfaceSignature(t string) string {

	var (
		jsonPkg = p.NewImport(jsonPkgPath)
		ctxPkg  = p.NewImport(ctxPkgPath)
	)

	return fmt.Sprintf(`interface{}(&%s{}).(interface { AtlasJSONValidate(%s.Context, %s.RawMessage, string) (%s.RawMessage, error) })`, t, ctxPkg.Use(), jsonPkg.Use(), jsonPkg.Use())

}

func (p *Plugin) renderAnnotator() {

	var (
		httpPkg     = p.NewImport(httpPkgPath)
		ctxPkg      = p.NewImport(ctxPkgPath)
		bytesPkg    = p.NewImport(bytesPkgPath)
		ioutilPkg   = p.NewImport(ioutilPkgPath)
		metadataPkg = p.NewImport(metadataPkgPath)
		runtimePkg  = p.NewImport(runtimePkgPath)
	)

	p.P(`// AtlasValidateAnnotator parses JSON input and validates unknown fields`)
	p.P(`// based on 'allow_unknown_fields' options specified in proto file.`)
	p.P(`func AtlasValidateAnnotator(ctx `, ctxPkg.Use(), `.Context, r *`, httpPkg.Use(), `.Request) `, metadataPkg.Use(), `.MD {`)
	p.P(`md := make(`, metadataPkg.Use(), `.MD)`)

	p.P(`for _, v := range validate_Patterns {`)
	p.P(`if r.Method == v.httpMethod && `, runtimePkg.Use(), `.PatternMatch(v.pattern, r.URL.Path) {`)
	p.P(`var b []byte`)
	p.P(`var err error`)
	p.P(`if b, err = `, ioutilPkg.Use(), `.ReadAll(r.Body); err != nil {`)
	p.P(`md.Set("Atlas-Validation-Error", "invalid value: unable to parse body")`)
	p.P(`return md`)
	p.P(`}`)
	p.P(`r.Body = `, ioutilPkg.Use(), `.NopCloser(`, bytesPkg.Use(), `.NewReader(b))`)
	p.P(`ctx := `, ctxPkg.Use(), `.WithValue(`, ctxPkg.Use(), `.WithValue(`, ctxPkg.Use(), `.Background(), `, runtimePkg.Use(), `.HTTPMethodContextKey, r.Method), `, runtimePkg.Use(), `.AllowUnknownContextKey, v.allowUnknown)`)
	p.P(`if err = v.validator(ctx, b); err != nil {`)
	p.P(`md.Set("Atlas-Validation-Error", err.Error())`)
	p.P(`}`)
	p.P(`break`)
	p.P(`}`)
	p.P(`}`)
	p.P(`return md`)
	p.P(`}`)
	p.P()
}

//Return methods to which field marked as denied
func (p *Plugin) GetDeniedMethods(options []av_opts.AtlasValidateFieldOption_Operation) []string {
	httpMethods := make(map[string]struct{}, 0)
	for _, op := range options {
		switch op {
		case av_opts.AtlasValidateFieldOption_create:
			httpMethods["POST"] = struct{}{}
		case av_opts.AtlasValidateFieldOption_update:
			httpMethods["PATCH"] = struct{}{}
		case av_opts.AtlasValidateFieldOption_replace:
			httpMethods["PUT"] = struct{}{}
		}
	}

	uniqueMethods := make([]string, 0)
	for m := range httpMethods {
		uniqueMethods = append(uniqueMethods, m)
	}

	sort.StringSlice(uniqueMethods).Sort()
	return uniqueMethods
}

//Return methods to which field marked as required
func (p *Plugin) GetRequiredMethods(options []av_opts.AtlasValidateFieldOption_Operation) []string {
	requiredMethods := make(map[string]struct{}, 0)
	for _, op := range options {
		switch op {
		case av_opts.AtlasValidateFieldOption_create:
			requiredMethods["POST"] = struct{}{}
		case av_opts.AtlasValidateFieldOption_update:
			requiredMethods["PATCH"] = struct{}{}
		case av_opts.AtlasValidateFieldOption_replace:
			requiredMethods["PUT"] = struct{}{}
		}
	}

	uniqueMethods := make([]string, 0)
	for m := range requiredMethods {
		uniqueMethods = append(uniqueMethods, m)
	}

	sort.StringSlice(uniqueMethods).Sort()
	return uniqueMethods
}

func (p *Plugin) generateValidateRequired(md *descriptor.DescriptorProto, t string) {

	var (
		fmtPkg     = p.NewImport(fmtPkgPath)
		ctxPkg     = p.NewImport(ctxPkgPath)
		jsonPkg    = p.NewImport(jsonPkgPath)
		runtimePkg = p.NewImport(runtimePkgPath)
	)

	requiredFields := make(map[string][]string)
	for _, fd := range md.GetField() {
		if fExt, err := proto.GetExtension(fd.Options, av_opts.E_Field); err == nil && fExt != nil {
			favOpt := fExt.(*av_opts.AtlasValidateFieldOption)
			methods := p.GetRequiredMethods(favOpt.GetRequired())
			if len(methods) == 0 {
				continue
			}
			requiredFields[fd.GetName()] = methods
		}
	}

	p.P(`func validate_required_Object_`, t, `(ctx `, ctxPkg.Use(), `.Context, v map[string]`, jsonPkg.Use(), `.RawMessage, path string) error {`)
	p.P(`method := `, runtimePkg.Use(), `.HTTPMethodFromContext(ctx)`)
	p.P(`_ = method`)

	for fn, methods := range requiredFields {
		if len(methods) == 3 {
			p.P(`if _, ok := v["`, fn, `"]; !ok {`)
			p.P(`path = `, runtimePkg.Use(), `.JoinPath(path, "`, fn, `")`)
			p.P(`return `, fmtPkg.Use(), `.Errorf("field %q is required for %q operation.", path, method)`)
			p.P(`}`)
		} else {
			cond := strings.Join(methods, `" || method == "`)
			p.P(`if _, ok := v["`, fn, `"]; !ok && (method == "`, cond, `") {`)
			p.P(`path = `, runtimePkg.Use(), `.JoinPath(path, "`, fn, `")`)
			p.P(`return `, fmtPkg.Use(), `.Errorf("field %q is required for %q operation.", path, method)`)
			p.P(`}`)
		}
	}
	p.P(`return nil`)
	p.P(`}`)
}
