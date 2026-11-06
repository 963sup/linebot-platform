// Package miniapp serves the LIFF Attendance client and maps its HTTP API to
// the consumer-owned Attendance Application contract.
package miniapp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
)

const maxRequestBytes = 4 << 10

//go:embed assets/*
var assets embed.FS

type Config struct {
	LIFFID        string
	AllowedOrigin string
}

type Handler struct {
	actions       attendance.Actions
	liffID        string
	allowedOrigin string
}

func New(config Config, actions attendance.Actions) (*Handler, error) {
	if actions == nil || strings.TrimSpace(config.LIFFID) == "" || config.LIFFID != strings.TrimSpace(config.LIFFID) {
		return nil, errors.New("miniapp requires a LIFF ID and Attendance actions")
	}
	origin, err := url.Parse(config.AllowedOrigin)
	localHTTP := origin != nil && origin.Scheme == "http" && (origin.Hostname() == "localhost" || origin.Hostname() == "127.0.0.1")
	if err != nil || origin == nil || (origin.Scheme != "https" && !localHTTP) || origin.Host == "" || origin.User != nil ||
		config.AllowedOrigin != origin.Scheme+"://"+origin.Host {
		return nil, errors.New("miniapp allowed origin must be exact HTTPS or loopback HTTP")
	}
	return &Handler{actions: actions, liffID: config.LIFFID, allowedOrigin: origin.String()}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	secureHeaders(w)
	switch r.URL.Path {
	case "/miniapp":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		http.Redirect(w, r, "/miniapp/", http.StatusPermanentRedirect)
	case "/miniapp/":
		h.serveAsset(w, r, "assets/index.html", "text/html; charset=utf-8")
	case "/miniapp/config.json":
		h.serveConfig(w, r)
	case "/miniapp/assets/app.css":
		h.serveAsset(w, r, "assets/app.css", "text/css; charset=utf-8")
	case "/miniapp/assets/app.js":
		h.serveAsset(w, r, "assets/app.js", "text/javascript; charset=utf-8")
	case "/miniapp/api/v1/attendance/enrollments":
		h.enroll(w, r)
	case "/miniapp/api/v1/attendance/options":
		h.listOptions(w, r)
	case "/miniapp/api/v1/attendance/punches":
		h.punch(w, r)
	default:
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	}
}

func secureHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' https://static.line-scdn.net; style-src 'self'; connect-src 'self' https://api.line.me; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("Permissions-Policy", "geolocation=(self)")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}

func (h *Handler) serveAsset(w http.ResponseWriter, r *http.Request, name, contentType string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	data, err := assets.ReadFile(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) serveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		LIFFID string `json:"liffId"`
	}{h.liffID})
}

type enrollmentRequest struct {
	EnrollmentCode string `json:"enrollmentCode"`
}

func (h *Handler) enroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	token, ok := h.authorize(w, r)
	if !ok {
		return
	}
	var input enrollmentRequest
	if !decodeStrictJSON(w, r, &input) {
		return
	}
	result, err := h.actions.Enroll(r.Context(), attendance.EnrollCommand{IDToken: token, Code: input.EnrollmentCode})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	var httpStatus int
	switch result.Status {
	case attendance.EnrollCreated:
		httpStatus = http.StatusCreated
	case attendance.EnrollAlreadyCompleted:
		httpStatus = http.StatusOK
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
		return
	}
	writeJSON(w, httpStatus, struct {
		Status string `json:"status"`
	}{string(result.Status)})
}

func (h *Handler) listOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	token, ok := h.authorize(w, r)
	if !ok {
		return
	}
	result, err := h.actions.ListOptions(r.Context(), attendance.OptionsCommand{IDToken: token})
	if err != nil {
		if errors.Is(err, attendance.ErrNotEnrolled) {
			writeJSON(w, http.StatusOK, optionsResponse{EnrollmentRequired: true, Options: []optionResponse{}})
			return
		}
		writeApplicationError(w, err)
		return
	}
	options := make([]optionResponse, 0, len(result.Options))
	for _, option := range result.Options {
		kinds := make([]string, len(option.AllowedKinds))
		for i, kind := range option.AllowedKinds {
			kinds[i] = string(kind)
		}
		options = append(options, optionResponse{ShiftInstanceID: option.ShiftInstanceID, Label: option.Label, WorkplaceLabel: option.WorkplaceLabel, AllowedKinds: kinds})
	}
	writeJSON(w, http.StatusOK, optionsResponse{Options: options})
}

type optionsResponse struct {
	EnrollmentRequired bool             `json:"enrollmentRequired"`
	Options            []optionResponse `json:"options"`
}

type optionResponse struct {
	ShiftInstanceID string   `json:"shiftInstanceId"`
	Label           string   `json:"label"`
	WorkplaceLabel  string   `json:"workplaceLabel"`
	AllowedKinds    []string `json:"allowedKinds"`
}

type punchRequest struct {
	ShiftInstanceID string      `json:"shiftInstanceId"`
	Kind            string      `json:"kind"`
	Latitude        json.Number `json:"latitude"`
	Longitude       json.Number `json:"longitude"`
	AccuracyMeters  json.Number `json:"accuracyMeters"`
}

func (h *Handler) punch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	token, ok := h.authorize(w, r)
	if !ok {
		return
	}
	var input punchRequest
	if !decodeStrictJSON(w, r, &input) {
		return
	}
	latitude, latErr := finiteFloat(input.Latitude)
	longitude, lonErr := finiteFloat(input.Longitude)
	accuracy, accuracyErr := finiteFloat(input.AccuracyMeters)
	if latErr != nil || lonErr != nil || accuracyErr != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
		return
	}
	result, err := h.actions.Punch(r.Context(), attendance.PunchCommand{
		IDToken: token, ShiftInstanceID: input.ShiftInstanceID, Kind: attendance.PunchKind(input.Kind),
		Location: attendance.LocationFix{Latitude: latitude, Longitude: longitude, AccuracyMeters: accuracy},
	})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	var httpStatus int
	switch result.Status {
	case attendance.PunchRecorded:
		httpStatus = http.StatusCreated
	case attendance.PunchAlreadyRecorded:
		httpStatus = http.StatusOK
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
		return
	}
	writeJSON(w, httpStatus, struct {
		AttendanceID string   `json:"attendanceId"`
		PunchID      string   `json:"punchId"`
		Kind         string   `json:"kind"`
		RecordedAt   string   `json:"recordedAt"`
		Status       string   `json:"status"`
		ReviewState  string   `json:"reviewState"`
		Reasons      []string `json:"reasons"`
	}{result.AttendanceID, result.PunchID, string(result.Kind), result.RecordedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), string(result.Status), result.ReviewState, nonNilStrings(result.ReviewReasons)})
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) (string, bool) {
	origins := r.Header.Values("Origin")
	originAllowed := exactHeader(origins, h.allowedOrigin)
	// Browsers may omit Origin on a same-origin GET. In that case Fetch
	// Metadata is mandatory; mutations still require the exact Origin header.
	if len(origins) == 0 && r.Method == http.MethodGet {
		originAllowed = exactHeader(r.Header.Values("Sec-Fetch-Site"), "same-origin")
	}
	if !originAllowed || !sameOriginFetch(r.Header.Values("Sec-Fetch-Site")) {
		writeError(w, http.StatusForbidden, "forbidden", "request origin is not allowed")
		return "", false
	}
	token, ok := bearerToken(r.Header.Values("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return "", false
	}
	return token, true
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	contentTypes := r.Header.Values("Content-Type")
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if len(contentTypes) != 1 || err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "content type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		writeDecodeError(w, err)
		return false
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		writeDecodeError(w, err)
		return false
	}
	return true
}

func bearerToken(values []string) (string, bool) {
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	return token, token != "" && token == strings.TrimSpace(token) && !strings.ContainsAny(token, "\t ,") && len(token) <= 4096
}

func exactHeader(values []string, want string) bool { return len(values) == 1 && values[0] == want }
func sameOriginFetch(values []string) bool {
	return len(values) == 0 || exactHeader(values, "same-origin")
}

func finiteFloat(number json.Number) (float64, error) {
	if number == "" {
		return 0, errors.New("missing number")
	}
	value, err := strconv.ParseFloat(number.String(), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("invalid number")
	}
	return value, nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds 4096 bytes")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_request", "request body is invalid")
}

func writeApplicationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, attendance.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_request", "request is invalid")
	case errors.Is(err, attendance.ErrUnauthenticated):
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
	case errors.Is(err, attendance.ErrEnrollmentInvalid):
		writeError(w, http.StatusNotFound, "enrollment_invalid", "enrollment code is invalid")
	case errors.Is(err, attendance.ErrNotEnrolled):
		writeError(w, http.StatusForbidden, "enrollment_required", "attendance enrollment is required")
	case errors.Is(err, attendance.ErrNotAssigned):
		writeError(w, http.StatusForbidden, "not_assigned", "shift is not assigned")
	case errors.Is(err, attendance.ErrSubmissionClosed):
		writeError(w, http.StatusConflict, "submission_closed", "attendance submission is closed")
	case errors.Is(err, attendance.ErrConflict):
		writeError(w, http.StatusConflict, "attendance_conflict", "attendance state has changed")
	case errors.Is(err, attendance.ErrIdentityUnavailable), errors.Is(err, attendance.ErrRepositoryUnavailable):
		writeError(w, http.StatusServiceUnavailable, "attendance_unavailable", "attendance service is unavailable")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "request_cancelled", "request was cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "deadline_exceeded", "request deadline exceeded")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var envelope errorEnvelope
	envelope.Error.Code, envelope.Error.Message = code, message
	writeJSON(w, status, envelope)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
