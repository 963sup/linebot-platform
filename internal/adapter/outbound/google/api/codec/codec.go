// Package codec contains provider-neutral JSON and error boundaries for the
// statically enumerated Google Calendar and Google Tasks SDK adapters.
package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"google.golang.org/api/googleapi"
)

// RemoteError deliberately never contains user data, authorization headers,
// or Google's raw error body.
type RemoteError struct {
	API string
	Operation string
	Status int
	OutcomeUnknown bool
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("%s %s failed (HTTP %d)",e.API,e.Operation,e.Status)
}

func Wrap(api, operation string, err error) error {
	if err == nil {return nil}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {return err}
	var remote *googleapi.Error
	if errors.As(err,&remote) {
		return &RemoteError{API:api,Operation:operation,Status:remote.Code}
	}
	return &RemoteError{API:api,Operation:operation,OutcomeUnknown:true}
}

func Validate(req outbound.WorkspaceRequest, expectsBody bool, required, allowed []string) error {
	if req.Operation == "" {return errors.New("workspace operation is required")}
	if len(req.Parameters) != len(required) {
		return errors.New("required workspace parameters are missing or unexpected")
	}
	expected:=map[string]bool{}
	for _,name:=range required {expected[name]=true}
	for name,raw:=range req.Parameters{
		if !expected[name] || !json.Valid(raw){return errors.New("unknown or invalid required parameter")}
	}
	permitted:=map[string]bool{}
	for _,name:=range allowed {permitted[name]=true}
	for name,raw:=range req.Options{
		if !permitted[name] || !json.Valid(raw){return errors.New("unknown or invalid workspace option")}
	}
	if expectsBody {
		body:=bytes.TrimSpace(req.Body)
		if len(body)==0 || body[0]!='{' || !json.Valid(body) || len(body)>1<<20 {
			return errors.New("request JSON object is required")
		}
	} else if len(req.Body)!=0 {
		return errors.New("unexpected workspace request body")
	}
	return nil
}

func Required[T any](params map[string]json.RawMessage,name string)(T,error) {
	var value T
	raw,ok:=params[name]
	if !ok || json.Unmarshal(raw,&value)!=nil {return value,errors.New("invalid required workspace parameter")}
	switch v:=any(value).(type) {
	case string:
		if strings.TrimSpace(v)=="" || v!=strings.TrimSpace(v) {return value,errors.New("empty or malformed workspace identifier")}
	}
	return value,nil
}

func Optional[T any](options map[string]json.RawMessage,name string)(T,error){
	var value T
	if err:=json.Unmarshal(options[name],&value);err!=nil{return value,errors.New("invalid workspace option")}
	return value,nil
}

func Body[T any](data json.RawMessage)(*T,error) {
	var body T
	if err:=json.Unmarshal(data,&body);err!=nil {return nil,errors.New("invalid workspace request body")}
	return &body,nil
}

func Encode(value any)(json.RawMessage,error) {
	if value == nil {return nil,nil}
	data,err:=json.Marshal(value)
	if err!=nil{return nil,errors.New("unable to encode workspace response")}
	return json.RawMessage(data),nil
}

func UnsupportedOperation() error {return errors.New("unsupported Google Workspace SDK operation")}
