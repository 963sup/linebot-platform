// Package mediatransfer owns explicit binary-content transfer workflows.
package mediatransfer

import (
	"context"
	"errors"
	"strings"
)

type ContentVariant string

const (
	Original ContentVariant = "original"
	Preview  ContentVariant = "preview"
)

type PreparationStatus string

const (
	PreparationProcessing PreparationStatus = "processing"
	PreparationReady      PreparationStatus = "ready"
	PreparationFailed     PreparationStatus = "failed"
)

type BinaryContent struct {
	Data        []byte
	ContentType string
}

type Gateway interface {
	FetchMessageContent(context.Context, string, ContentVariant) (BinaryContent, error)
	MessageContentStatus(context.Context, string) (PreparationStatus, error)
	FetchPresentationImage(context.Context, string) (BinaryContent, error)
	StorePresentationImage(context.Context, string, BinaryContent) error
}

type Service struct{ gateway Gateway }

func New(gateway Gateway) (*Service, error) {
	if gateway == nil {
		return nil, errors.New("media gateway is required")
	}
	return &Service{gateway: gateway}, nil
}

func (s *Service) MessageContent(ctx context.Context, messageID string, variant ContentVariant) (BinaryContent, error) {
	if err := validateContextAndID(ctx, messageID); err != nil {
		return BinaryContent{}, err
	}
	if variant != Original && variant != Preview {
		return BinaryContent{}, errors.New("unsupported content variant")
	}
	return s.gateway.FetchMessageContent(ctx, messageID, variant)
}

func (s *Service) MessageContentStatus(ctx context.Context, messageID string) (PreparationStatus, error) {
	if err := validateContextAndID(ctx, messageID); err != nil {
		return "", err
	}
	return s.gateway.MessageContentStatus(ctx, messageID)
}

func (s *Service) PresentationImage(ctx context.Context, presentationID string) (BinaryContent, error) {
	if err := validateContextAndID(ctx, presentationID); err != nil {
		return BinaryContent{}, err
	}
	return s.gateway.FetchPresentationImage(ctx, presentationID)
}

func (s *Service) StorePresentationImage(ctx context.Context, presentationID string, content BinaryContent) error {
	if err := validateContextAndID(ctx, presentationID); err != nil {
		return err
	}
	if len(content.Data) == 0 || (content.ContentType != "image/png" && content.ContentType != "image/jpeg") {
		return errors.New("a non-empty PNG or JPEG image is required")
	}
	return s.gateway.StorePresentationImage(ctx, presentationID, content)
}

func validateContextAndID(ctx context.Context, id string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" || id != strings.TrimSpace(id) {
		return errors.New("content identifier is required")
	}
	return nil
}
