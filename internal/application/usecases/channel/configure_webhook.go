package channel

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
)

type WebhookConfigurator struct { client outbound.WebhookConfigurer }
func NewWebhookConfigurator(client outbound.WebhookConfigurer) (*WebhookConfigurator,error) {
	if client==nil { return nil,errors.New("webhook configurer required") }
	return &WebhookConfigurator{client:client},nil
}

func ValidateWebhookURL(raw string) error {
	if raw=="" || raw!=strings.TrimSpace(raw) { return errors.New("webhook URL required") }
	u,err:=url.Parse(raw)
	if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.Fragment!="" || u.RawQuery!="" || u.Opaque!="" {
		return errors.New("webhook URL must be an absolute HTTPS URL without credentials, query or fragment")
	}
	return nil
}

func (s *WebhookConfigurator) Set(ctx context.Context, endpoint string) error {
	if ctx==nil {return errors.New("context required")}
	if err:=ctx.Err();err!=nil{return err}
	if err:=ValidateWebhookURL(endpoint);err!=nil{return err}
	return s.client.SetWebhookEndpoint(ctx,endpoint)
}
func (s *WebhookConfigurator) Test(ctx context.Context, endpoint string) (outbound.WebhookEndpointTest,error) {
	if ctx==nil {return outbound.WebhookEndpointTest{},errors.New("context required")}
	if err:=ctx.Err();err!=nil{return outbound.WebhookEndpointTest{},err}
	if err:=ValidateWebhookURL(endpoint);err!=nil{return outbound.WebhookEndpointTest{},err}
	return s.client.TestWebhookEndpoint(ctx,endpoint)
}
