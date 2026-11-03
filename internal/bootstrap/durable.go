package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	linein "github.com/963sup/linebot-platform/internal/adapter/inbound/http/line"
	queueout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/cloudtasks"
	storeout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	"github.com/963sup/linebot-platform/internal/application/usecases/ingestion"
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
	queueClient, err := queueinfra.NewClient(ctx)
	if err != nil {
		_ = db.Close()
		return err
	}
	store, err := storeout.NewStore(db)
	if err != nil {
		_ = queueClient.Close()
		_ = db.Close()
		return err
	}
	queue, err := queueout.NewQueue(queueClient, cfg.Durable.QueueName, cfg.Durable.WorkerURL,
		cfg.Durable.OIDCAudience, cfg.Durable.InvokerEmail)
	if err != nil {
		_ = queueClient.Close()
		_ = db.Close()
		return err
	}
	ingestor, err := ingestion.NewIngestor(store, queue)
	if err != nil {
		_ = queueClient.Close()
		_ = db.Close()
		return err
	}
	mux.Handle("/webhook/line", linein.NewDurableHandler(cfg.ChannelSecret, cfg.Durable.BotDestination, ingestor, logger))
	rt.closers = append(rt.closers, queueClient.Close, db.Close)
	return nil
}
