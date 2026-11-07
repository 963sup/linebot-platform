package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	linein "github.com/963sup/linebot-platform/internal/adapter/inbound/http/line"
	queueout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/cloudtasks"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	ingestion "github.com/963sup/linebot-platform/internal/application/usecases/messaging-delivery"
	queueinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	storeinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
)

// configureDurableWebhook wires only the Cloud Tasks queue and Firestore store.
// It does not invoke LINE Messaging or initialize unrelated AI/Workspace SDKs.
func configureDurableWebhook(cfg Config, logger *slog.Logger, mux *http.ServeMux, rt *WebhookRuntime) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := storeinfra.NewClient(ctx, cfg.Durable.ProjectID)
	if err != nil {
		return err
	}
	rt.closers = append(rt.closers, runtimeCloser{name: "Firestore client", close: db.Close})
	queueClient, err := queueinfra.NewClient(ctx)
	if err != nil {
		return err
	}
	rt.closers = append(rt.closers, runtimeCloser{name: "Cloud Tasks client", close: queueClient.Close})
	store, err := storeout.NewStore(db)
	if err != nil {
		return err
	}
	namespace, err := storeout.EventKeyNamespace(ctx, db, cfg.Durable.BotUserID)
	if err != nil {
		return err
	}
	queue, err := queueout.NewQueue(queueClient, cfg.Durable.QueueName, cfg.Durable.WorkerURL,
		cfg.Durable.OIDCAudience, cfg.Durable.InvokerEmail)
	if err != nil {
		return err
	}
	ingestor, err := ingestion.NewIngestor(store, queue, namespace)
	if err != nil {
		return err
	}
	policy, err := ingestion.NewReplyPolicy(cfg.MiniAppEntryURL)
	if err != nil {
		return err
	}
	mux.Handle("/webhook/line", linein.NewDurableHandler(cfg.Secrets.MessagingAPIChannelSecret, cfg.Durable.BotUserID, ingestor, policy, logger))
	return nil
}
