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

	workspace "github.com/963sup/linebot-platform/internal/application/usecases/workspace-automation"
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

// Request is the adapter-owned decoded representation used to map the
// provider-neutral application command onto generated Google SDK calls.
// Untyped JSON is deliberately confined to this outbound boundary.
type Request struct {
	Operation  string
	Parameters map[string]any
	Options    map[string]any
	Body       any
}

// Decode maps the consumer-owned application contract into adapter-local JSON
// values. Plain scalar strings are accepted for ergonomic callers; valid JSON
// remains byte-for-byte compatible with the previous CLI contract.
func Decode(command workspace.OperationCommand) (Request, error) {
	request := Request{
		Operation:  string(command.Operation),
		Parameters: make(map[string]any, len(command.Parameters)),
		Options:    make(map[string]any, len(command.Options)),
	}
	for _, parameter := range command.Parameters {
		if _, exists := request.Parameters[parameter.Name]; exists {
			return Request{}, errors.New("duplicate workspace parameter")
		}
		request.Parameters[parameter.Name] = scalarJSON(parameter.Value)
	}
	for _, option := range command.Options {
		if _, exists := request.Options[option.Name]; exists {
			return Request{}, errors.New("duplicate workspace option")
		}
		if len(option.Values) == 1 {
			request.Options[option.Name] = scalarJSON(option.Values[0])
			continue
		}
		values := make([]string, len(option.Values))
		for index, value := range option.Values {
			var decoded string
			if json.Unmarshal([]byte(value), &decoded) == nil {
				values[index] = decoded
			} else {
				values[index] = value
			}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return Request{}, errors.New("invalid workspace option")
		}
		request.Options[option.Name] = json.RawMessage(encoded)
	}
	if len(command.Payload) != 0 {
		if !json.Valid(command.Payload) {
			return Request{}, errors.New("invalid workspace request body")
		}
		request.Body = json.RawMessage(append([]byte(nil), command.Payload...))
	}
	return request, nil
}

func scalarJSON(value string) json.RawMessage {
	if json.Valid([]byte(value)) {
		return json.RawMessage(append([]byte(nil), value...))
	}
	encoded, _ := json.Marshal(value)
	return json.RawMessage(encoded)
}

// Command is used by adapter tests and trusted inbound mapping code to build
// the typed application contract from already-decoded JSON values.
func Command(request Request) (workspace.OperationCommand, error) {
	command := workspace.OperationCommand{Operation: workspace.Operation(request.Operation)}
	for name, value := range request.Parameters {
		encoded, err := json.Marshal(value)
		if err != nil {
			return workspace.OperationCommand{}, errors.New("invalid workspace parameter")
		}
		command.Parameters = append(command.Parameters, workspace.Parameter{Name: name, Value: string(encoded)})
	}
	for name, value := range request.Options {
		encoded, err := json.Marshal(value)
		if err != nil {
			return workspace.OperationCommand{}, errors.New("invalid workspace option")
		}
		command.Options = append(command.Options, workspace.Option{Name: name, Values: []string{string(encoded)}})
	}
	if !isNil(request.Body) {
		encoded, err := json.Marshal(request.Body)
		if err != nil {
			return workspace.OperationCommand{}, errors.New("invalid workspace request body")
		}
		command.Payload = encoded
	}
	return command, nil
}

func Validate(req Request, expectsBody bool, required, allowed []string) error {
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

func Encode(value any) (workspace.OperationResult, error) {
	if value == nil {
		return workspace.OperationResult{}, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return workspace.OperationResult{}, errors.New("unable to encode workspace response")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var result any
	if err := decoder.Decode(&result); err != nil {
		return workspace.OperationResult{}, errors.New("unable to decode workspace response")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return workspace.OperationResult{}, errors.New("invalid workspace response")
	}
	canonical, err := json.Marshal(result)
	if err != nil {
		return workspace.OperationResult{}, errors.New("unable to encode workspace response")
	}
	return workspace.OperationResult{Payload: canonical}, nil
}

func UnsupportedOperation() error { return errors.New("unsupported Google Workspace SDK operation") }
