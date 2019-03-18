// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: example/external/external.proto

package external // import "github.com/infobloxopen/protoc-gen-atlas-validate/example/external"

import http "net/http"
import json "encoding/json"
import context "context"
import fmt "fmt"
import ioutil "io/ioutil"
import metadata "google.golang.org/grpc/metadata"
import bytes "bytes"
import runtime "github.com/infobloxopen/protoc-gen-atlas-validate/runtime"
import runtime1 "github.com/grpc-ecosystem/grpc-gateway/runtime"
import proto "github.com/gogo/protobuf/proto"
import math "math"
import _ "github.com/infobloxopen/protoc-gen-atlas-validate/options"
import _ "google.golang.org/genproto/googleapis/api/annotations"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// validate_Object_ExternalUser function validates a JSON for a given object.
func validate_Object_ExternalUser(ctx context.Context, r json.RawMessage, path string) (err error) {
	if hook, ok := interface{}(&ExternalUser{}).(interface {
		AtlasJSONValidate(context.Context, json.RawMessage, string) (json.RawMessage, error)
	}); ok {
		if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {
			return err
		}
	}
	var v map[string]json.RawMessage
	if err = json.Unmarshal(r, &v); err != nil {
		return fmt.Errorf("invalid value for %q: expected object.", path)
	}
	allowUnknown := runtime.AllowUnknownFromContext(ctx)

	if err = validate_required_Object_ExternalUser(ctx, v, path); err != nil {
		return err
	}

	for k, _ := range v {
		switch k {
		case "id":
		case "name":
		case "address":
			if v[k] == nil {
				continue
			}
			vv := v[k]
			vvPath := runtime.JoinPath(path, k)
			if err = validate_Object_ExternalAddress(ctx, vv, vvPath); err != nil {
				return err
			}
		default:
			if !allowUnknown {
				return fmt.Errorf("unknown field %q.", runtime.JoinPath(path, k))
			}
		}
	}
	return nil
}

// AtlasValidateJSON function validates a JSON for object ExternalUser.
func (_ *ExternalUser) AtlasValidateJSON(ctx context.Context, r json.RawMessage, path string) (err error) {
	if hook, ok := interface{}(&ExternalUser{}).(interface {
		AtlasJSONValidate(context.Context, json.RawMessage, string) (json.RawMessage, error)
	}); ok {
		if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {
			return err
		}
	}
	return validate_Object_ExternalUser(ctx, r, path)
}

func validate_required_Object_ExternalUser(ctx context.Context, v map[string]json.RawMessage, path string) error {
	method := runtime.HTTPMethodFromContext(ctx)
	_ = method
	return nil
}

// validate_Object_ExternalUser_Parent function validates a JSON for a given object.
func validate_Object_ExternalUser_Parent(ctx context.Context, r json.RawMessage, path string) (err error) {
	if hook, ok := interface{}(&ExternalUser_Parent{}).(interface {
		AtlasJSONValidate(context.Context, json.RawMessage, string) (json.RawMessage, error)
	}); ok {
		if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {
			return err
		}
	}
	var v map[string]json.RawMessage
	if err = json.Unmarshal(r, &v); err != nil {
		return fmt.Errorf("invalid value for %q: expected object.", path)
	}
	allowUnknown := runtime.AllowUnknownFromContext(ctx)

	if err = validate_required_Object_ExternalUser_Parent(ctx, v, path); err != nil {
		return err
	}

	for k, _ := range v {
		switch k {
		case "name":
		default:
			if !allowUnknown {
				return fmt.Errorf("unknown field %q.", runtime.JoinPath(path, k))
			}
		}
	}
	return nil
}

// AtlasValidateJSON function validates a JSON for object ExternalUser_Parent.
func (_ *ExternalUser_Parent) AtlasValidateJSON(ctx context.Context, r json.RawMessage, path string) (err error) {
	if hook, ok := interface{}(&ExternalUser_Parent{}).(interface {
		AtlasJSONValidate(context.Context, json.RawMessage, string) (json.RawMessage, error)
	}); ok {
		if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {
			return err
		}
	}
	return validate_Object_ExternalUser_Parent(ctx, r, path)
}

func validate_required_Object_ExternalUser_Parent(ctx context.Context, v map[string]json.RawMessage, path string) error {
	method := runtime.HTTPMethodFromContext(ctx)
	_ = method
	return nil
}

// validate_Object_ExternalAddress function validates a JSON for a given object.
func validate_Object_ExternalAddress(ctx context.Context, r json.RawMessage, path string) (err error) {
	if hook, ok := interface{}(&ExternalAddress{}).(interface {
		AtlasJSONValidate(context.Context, json.RawMessage, string) (json.RawMessage, error)
	}); ok {
		if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {
			return err
		}
	}
	var v map[string]json.RawMessage
	if err = json.Unmarshal(r, &v); err != nil {
		return fmt.Errorf("invalid value for %q: expected object.", path)
	}
	allowUnknown := runtime.AllowUnknownFromContext(ctx)

	if err = validate_required_Object_ExternalAddress(ctx, v, path); err != nil {
		return err
	}

	for k, _ := range v {
		switch k {
		case "country":
		case "state":
		case "city":
		case "zip":
		default:
			if !allowUnknown {
				return fmt.Errorf("unknown field %q.", runtime.JoinPath(path, k))
			}
		}
	}
	return nil
}

// AtlasValidateJSON function validates a JSON for object ExternalAddress.
func (_ *ExternalAddress) AtlasValidateJSON(ctx context.Context, r json.RawMessage, path string) (err error) {
	if hook, ok := interface{}(&ExternalAddress{}).(interface {
		AtlasJSONValidate(context.Context, json.RawMessage, string) (json.RawMessage, error)
	}); ok {
		if r, err = hook.AtlasJSONValidate(ctx, r, path); err != nil {
			return err
		}
	}
	return validate_Object_ExternalAddress(ctx, r, path)
}

func validate_required_Object_ExternalAddress(ctx context.Context, v map[string]json.RawMessage, path string) error {
	method := runtime.HTTPMethodFromContext(ctx)
	_ = method
	return nil
}

var validate_Patterns = []struct {
	pattern    runtime1.Pattern
	httpMethod string
	validator  func(context.Context, json.RawMessage) error
	// Included for introspection purpose.
	allowUnknown bool
}{
	// patterns for file example/external/external.proto

}

// AtlasValidateAnnotator parses JSON input and validates unknown fields
// based on 'allow_unknown_fields' options specified in proto file.
func AtlasValidateAnnotator(ctx context.Context, r *http.Request) metadata.MD {
	md := make(metadata.MD)
	for _, v := range validate_Patterns {
		if r.Method == v.httpMethod && runtime.PatternMatch(v.pattern, r.URL.Path) {
			var b []byte
			var err error
			if b, err = ioutil.ReadAll(r.Body); err != nil {
				md.Set("Atlas-Validation-Error", "invalid value: unable to parse body")
				return md
			}
			r.Body = ioutil.NopCloser(bytes.NewReader(b))
			ctx := context.WithValue(context.WithValue(context.Background(), runtime.HTTPMethodContextKey, r.Method), runtime.AllowUnknownContextKey, v.allowUnknown)
			if err = v.validator(ctx, b); err != nil {
				md.Set("Atlas-Validation-Error", err.Error())
			}
			break
		}
	}
	return md
}
