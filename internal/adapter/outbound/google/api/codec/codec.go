// Package codec contains provider-neutral JSON and error boundaries for the
// statically enumerated Google Calendar and Google Tasks SDK adapters.
package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/usecases/workspace"
	"google.golang.org/api/googleapi"
)

// RemoteError deliberately never contains user data, authorization headers,
// or Google's raw error body.
type RemoteError struct {
	API            string
	Operation      string
	Status         int
	OutcomeUnknown bool
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("%s %s failed (HTTP %d)", e.API, e.Operation, e.Status)
}

func Wrap(api, operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var remote *googleapi.Error
	if errors.As(err, &remote) {
		return &RemoteError{API: api, Operation: operation, Status: remote.Code}
	}
	return &RemoteError{API: api, Operation: operation, OutcomeUnknown: true}
}

func Validate(req workspace.Command, expectsBody bool, required, allowed []string) error {
	if req.Operation == "" {
		return errors.New("workspace operation is required")
	}
	if len(req.Parameters) != len(required) {
		return errors.New("required workspace parameters are missing or unexpected")
	}
	expected := map[string]bool{}
	for _, name := range required {
		expected[name] = true
	}
	for name := range req.Parameters {
		if !expected[name] {
			return errors.New("unknown or invalid required parameter")
		}
	}
	permitted := map[string]bool{}
	for _, name := range allowed {
		permitted[name] = true
	}
	for name := range req.Options {
		if !permitted[name] {
			return errors.New("unknown or invalid workspace option")
		}
	}
	if expectsBody {
		if isNil(req.Body) {
			return errors.New("request JSON object is required")
		}
	} else if !isNil(req.Body) {
		return errors.New("unexpected workspace request body")
	}
	return nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func Required[T any](params map[string]any, name string) (T, error) {
	var value T
	raw, ok := params[name]
	data, err := json.Marshal(raw)
	if !ok || err != nil || json.Unmarshal(data, &value) != nil {
		return value, errors.New("invalid required workspace parameter")
	}
	switch v := any(value).(type) {
	case string:
		if strings.TrimSpace(v) == "" || v != strings.TrimSpace(v) {
			return value, errors.New("empty or malformed workspace identifier")
		}
	}
	return value, nil
}

func Optional[T any](options map[string]any, name string) (T, error) {
	var value T
	data, err := json.Marshal(options[name])
	if err != nil || json.Unmarshal(data, &value) != nil {
		return value, errors.New("invalid workspace option")
	}
	return value, nil
}

func Body[T any](value any) (*T, error) {
	var body T
	data, err := json.Marshal(value)
	if err != nil || json.Unmarshal(data, &body) != nil {
		return nil, errors.New("invalid workspace request body")
	}
	return &body, nil
}

func Encode(value any) (workspace.Result, error) {
	if value == nil {
		return workspace.Result{}, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return workspace.Result{}, errors.New("unable to encode workspace response")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var result any
	if err := decoder.Decode(&result); err != nil {
		return workspace.Result{}, errors.New("unable to decode workspace response")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return workspace.Result{}, errors.New("invalid workspace response")
	}
	return workspace.Result{Value: result}, nil
}

func UnsupportedOperation() error { return errors.New("unsupported Google Workspace SDK operation") }
