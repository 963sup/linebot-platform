package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	openaisdk "github.com/openai/openai-go/v3"
	genaisdk "google.golang.org/genai"

	genaiinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/genai"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
	openaiinfra "github.com/963sup/linebot-platform/internal/infrastructure/openai"
)

// AIConfig enables only the explicitly configured provider clients. An empty
// configuration is valid so existing runtimes do not acquire an AI dependency.
type AIConfig struct {
	OpenAIAPIKey   string
	GeminiAPIKey   string
	VertexProject  string
	VertexLocation string
	Timeout        time.Duration
}

func (AIConfig) String() string   { return "[redacted AI configuration]" }
func (AIConfig) GoString() string { return "bootstrap.AIConfig{redacted}" }

// AIClients owns the optional native provider clients and their HTTP
// transports. It deliberately provides no routing or generation abstraction.
type AIClients struct {
	OpenAI *openaisdk.Client
	Gemini *genaisdk.Client
	Vertex *genaisdk.Client

	transports []*httpclient.Managed
}

// NewAIClients constructs configured native SDK clients without making a
// network request. Vertex AI uses ADC through the GenAI SDK and therefore does
// not receive an unauthenticated custom HTTP client.
func NewAIClients(ctx context.Context, cfg AIConfig) (*AIClients, error) {
	if ctx == nil {
		return nil, errors.New("AI context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	projectSet := strings.TrimSpace(cfg.VertexProject) != ""
	locationSet := strings.TrimSpace(cfg.VertexLocation) != ""
	if projectSet != locationSet {
		return nil, errors.New("Vertex AI project and location must be configured together")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = httpclient.DefaultTimeout
	}
	clients := &AIClients{}
	fail := func(err error) (*AIClients, error) {
		clients.Close()
		return nil, err
	}

	if strings.TrimSpace(cfg.OpenAIAPIKey) != "" {
		transport, err := httpclient.New(httpclient.Config{Timeout: timeout}, nil)
		if err != nil {
			return fail(err)
		}
		clients.transports = append(clients.transports, transport)
		client, err := openaiinfra.NewClient(cfg.OpenAIAPIKey, transport.Client)
		if err != nil {
			return fail(err)
		}
		clients.OpenAI = &client
	}
	if strings.TrimSpace(cfg.GeminiAPIKey) != "" {
		transport, err := httpclient.New(httpclient.Config{Timeout: timeout}, nil)
		if err != nil {
			return fail(err)
		}
		clients.transports = append(clients.transports, transport)
		client, err := genaiinfra.NewGeminiClient(ctx, cfg.GeminiAPIKey, transport.Client)
		if err != nil {
			return fail(err)
		}
		clients.Gemini = client
	}
	if projectSet {
		client, err := genaiinfra.NewVertexClient(ctx, cfg.VertexProject, cfg.VertexLocation)
		if err != nil {
			return fail(err)
		}
		clients.Vertex = client
	}
	return clients, nil
}

// Close releases the owned HTTP transports. It is safe to call repeatedly.
func (c *AIClients) Close() {
	if c == nil {
		return
	}
	for _, transport := range c.transports {
		transport.Close()
	}
}
