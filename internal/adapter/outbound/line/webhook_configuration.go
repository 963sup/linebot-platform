package line

import (
	"context"
	"errors"

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func (s *Sender) SetWebhookEndpoint(ctx context.Context, endpoint string) error {
	client,err:=s.clientFor(ctx)
	if err!=nil{return err}
	response,_,err:=client.SetWebhookEndpointWithHttpInfo(&messaging_api.SetWebhookEndpointRequest{Endpoint:endpoint})
	return lineWriteError("setWebhookEndpoint",response,err)
}

func (s *Sender) TestWebhookEndpoint(ctx context.Context, endpoint string) (outbound.WebhookEndpointTest,error) {
	client,err:=s.clientFor(ctx)
	if err!=nil{return outbound.WebhookEndpointTest{},err}
	response,result,err:=client.TestWebhookEndpointWithHttpInfo(&messaging_api.TestWebhookEndpointRequest{Endpoint:endpoint})
	if err!=nil{return outbound.WebhookEndpointTest{},lineWriteError("testWebhookEndpoint",response,err)}
	if result==nil{return outbound.WebhookEndpointTest{},errors.New("LINE webhook test response is empty")}
	return outbound.WebhookEndpointTest{Success:result.Success,StatusCode:result.StatusCode,Reason:result.Reason},nil
}
