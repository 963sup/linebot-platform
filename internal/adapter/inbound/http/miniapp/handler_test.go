package miniapp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/application/usecases/attendance"
)

const testOrigin = "https://miniapp.example.test"

type actionsStub struct {
	token       string
	enroll      attendance.EnrollResult
	options     attendance.OptionsResult
	punch       attendance.PunchResult
	err         error
	punchCmd    attendance.PunchCommand
	enrollCalls int
}

func (a *actionsStub) Enroll(_ context.Context, command attendance.EnrollCommand) (attendance.EnrollResult, error) {
	a.token, a.enrollCalls = command.IDToken, a.enrollCalls+1
	return a.enroll, a.err
}
func (a *actionsStub) ListOptions(_ context.Context, command attendance.OptionsCommand) (attendance.OptionsResult, error) {
	a.token = command.IDToken
	return a.options, a.err
}
func (a *actionsStub) Punch(_ context.Context, command attendance.PunchCommand) (attendance.PunchResult, error) {
	a.token, a.punchCmd = command.IDToken, command
	return a.punch, a.err
}

func newTestHandler(t *testing.T, actions attendance.Actions) *Handler {
	t.Helper()
	handler, err := New(Config{LIFFID: "123-test", AllowedOrigin: testOrigin}, actions)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	actions := &actionsStub{}
	for _, config := range []Config{{}, {LIFFID: " id ", AllowedOrigin: testOrigin}, {LIFFID: "id", AllowedOrigin: "http://example.test"}, {LIFFID: "id", AllowedOrigin: testOrigin + "/path"}} {
		if _, err := New(config, actions); err == nil {
			t.Fatalf("accepted config %+v", config)
		}
	}
	if _, err := New(Config{LIFFID: "id", AllowedOrigin: testOrigin}, nil); err == nil {
		t.Fatal("accepted nil actions")
	}
	for _, origin := range []string{"http://localhost:8080", "http://127.0.0.1:9090"} {
		if _, err := New(Config{LIFFID: "id", AllowedOrigin: origin}, actions); err != nil {
			t.Fatalf("rejected local origin: %v", err)
		}
	}
}

func TestAssetsExposeAttendanceWithoutPersistentBrowserTracking(t *testing.T) {
	handler := newTestHandler(t, &actionsStub{})
	for _, tc := range []struct{ path, contentType string }{{"/miniapp/", "text/html"}, {"/miniapp/assets/app.css", "text/css"}, {"/miniapp/assets/app.js", "text/javascript"}, {"/miniapp/config.json", "application/json"}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, testOrigin+tc.path, nil))
		if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), tc.contentType) {
			t.Fatalf("GET %s = %d %q", tc.path, response.Code, response.Header().Get("Content-Type"))
		}
		assertSafeResponse(t, response)
	}
	js := httptest.NewRecorder()
	handler.ServeHTTP(js, httptest.NewRequest(http.MethodGet, testOrigin+"/miniapp/assets/app.js", nil))
	for _, required := range []string{"attendance/enrollments", "attendance/options", "attendance/punches", "liff.getIDToken()", "getCurrentPosition", "enableHighAccuracy: false", "maximumAge: 0", "timeout: 10000", "clock_in", "clock_out", "enrollmentCode"} {
		if !strings.Contains(js.Body.String(), required) {
			t.Errorf("JavaScript missing %q", required)
		}
	}
	for _, forbidden := range []string{"location-validations", "getAccessToken", "localStorage", "sessionStorage", "watchPosition", "analytics", "google.maps", "leaflet"} {
		if strings.Contains(js.Body.String(), forbidden) {
			t.Errorf("JavaScript contains forbidden capability %q", forbidden)
		}
	}
}

func TestEnrollmentCreatedAndIdempotentStatuses(t *testing.T) {
	for _, tc := range []struct {
		enrollmentStatus attendance.EnrollStatus
		status           int
		body             string
	}{{attendance.EnrollCreated, 201, `"status":"enrolled"`}, {attendance.EnrollAlreadyCompleted, 200, `"status":"already_enrolled"`}} {
		actions := &actionsStub{enroll: attendance.EnrollResult{Status: tc.enrollmentStatus}}
		response := httptest.NewRecorder()
		newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", `{"enrollmentCode":"invite-code"}`))
		if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.body) || actions.token != "id-token" {
			t.Fatalf("response=%d %s token=%q", response.Code, response.Body.String(), actions.token)
		}
		assertSafeResponse(t, response)
	}
}

func TestUnknownEnrollmentStatusReturnsSafeInternalError(t *testing.T) {
	actions := &actionsStub{enroll: attendance.EnrollResult{Status: attendance.EnrollStatus("future_enrollment_status")}}
	response := httptest.NewRecorder()
	newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", `{"enrollmentCode":"invite-code"}`))

	wantBody := "{\"error\":{\"code\":\"internal_error\",\"message\":\"request could not be completed\"}}\n"
	if response.Code != http.StatusInternalServerError || response.Body.String() != wantBody {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "future_enrollment_status") {
		t.Fatalf("response exposed unknown status: %s", response.Body.String())
	}
	assertSafeResponse(t, response)
}

func TestOptionsAreAuthenticatedAndParticipantScoped(t *testing.T) {
	actions := &actionsStub{options: attendance.OptionsResult{Options: []attendance.AttendanceOption{{ShiftInstanceID: "shift-opaque", Label: "Morning", WorkplaceLabel: "Main workplace", AllowedKinds: []attendance.PunchKind{attendance.ClockIn}}}}}
	response := httptest.NewRecorder()
	newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodGet, "/miniapp/api/v1/attendance/options", ""))
	if response.Code != http.StatusOK || actions.token != "id-token" || !strings.Contains(response.Body.String(), `"shiftInstanceId":"shift-opaque"`) || !strings.Contains(response.Body.String(), `"workplaceLabel":"Main workplace"`) || !strings.Contains(response.Body.String(), `"allowedKinds":["clock_in"]`) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	assertSafeResponse(t, response)

	notEnrolled := httptest.NewRecorder()
	newTestHandler(t, &actionsStub{err: attendance.ErrNotEnrolled}).ServeHTTP(notEnrolled, apiRequest(http.MethodGet, "/miniapp/api/v1/attendance/options", ""))
	if notEnrolled.Code != http.StatusOK || !strings.Contains(notEnrolled.Body.String(), `"enrollmentRequired":true`) {
		t.Fatalf("not enrolled response=%d %s", notEnrolled.Code, notEnrolled.Body.String())
	}
}

func TestPunchCreatedAndReplayNeverReturnCoordinates(t *testing.T) {
	recordedAt := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		status attendance.PunchStatus
		want   int
	}{{attendance.PunchRecorded, 201}, {attendance.PunchAlreadyRecorded, 200}} {
		actions := &actionsStub{punch: attendance.PunchResult{AttendanceID: "attendance-opaque", PunchID: "punch-opaque", Kind: attendance.ClockIn, RecordedAt: recordedAt, Status: tc.status, ReviewState: "required", ReviewReasons: []string{"low_accuracy"}}}
		response := httptest.NewRecorder()
		body := `{"shiftInstanceId":"shift-opaque","kind":"clock_in","latitude":25.033,"longitude":121.565,"accuracyMeters":12.5}`
		newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/punches", body))
		if response.Code != tc.want || !strings.Contains(response.Body.String(), `"attendanceId":"attendance-opaque"`) || !strings.Contains(response.Body.String(), `"reviewState":"required"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
		for _, forbidden := range []string{"latitude", "longitude", "accuracyMeters", "25.033", "121.565", "12.5"} {
			if strings.Contains(response.Body.String(), forbidden) {
				t.Errorf("response leaked %q: %s", forbidden, response.Body.String())
			}
		}
		if actions.punchCmd.ShiftInstanceID != "shift-opaque" || actions.punchCmd.Location.AccuracyMeters != 12.5 {
			t.Fatalf("command mapping=%+v", actions.punchCmd)
		}
		assertSafeResponse(t, response)
	}
}

func TestUnknownPunchStatusReturnsSafeInternalErrorWithoutIDs(t *testing.T) {
	actions := &actionsStub{punch: attendance.PunchResult{
		AttendanceID: "attendance-secret",
		PunchID:      "punch-secret",
		Status:       attendance.PunchStatus("future_punch_status"),
	}}
	response := httptest.NewRecorder()
	body := `{"shiftInstanceId":"shift-opaque","kind":"clock_in","latitude":25.033,"longitude":121.565,"accuracyMeters":12.5}`
	newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/punches", body))

	wantBody := "{\"error\":{\"code\":\"internal_error\",\"message\":\"request could not be completed\"}}\n"
	if response.Code != http.StatusInternalServerError || response.Body.String() != wantBody {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	for _, forbidden := range []string{"attendance-secret", "punch-secret", "future_punch_status", "attendanceId", "punchId"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Errorf("response exposed %q: %s", forbidden, response.Body.String())
		}
	}
	assertSafeResponse(t, response)
}

func TestAttendanceAPISecurityAndStrictJSON(t *testing.T) {
	valid := `{"enrollmentCode":"code"}`
	tests := []struct {
		name   string
		mutate func(*http.Request)
		body   string
		status int
		code   string
	}{
		{"missing origin", func(r *http.Request) { r.Header.Del("Origin") }, valid, 403, "forbidden"},
		{"duplicate origin", func(r *http.Request) { r.Header.Add("Origin", testOrigin) }, valid, 403, "forbidden"},
		{"cross site", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, valid, 403, "forbidden"},
		{"missing bearer", func(r *http.Request) { r.Header.Del("Authorization") }, valid, 401, "unauthorized"},
		{"duplicate bearer", func(r *http.Request) { r.Header.Add("Authorization", "Bearer second") }, valid, 401, "unauthorized"},
		{"unknown field", nil, `{"enrollmentCode":"code","extra":true}`, 400, "invalid_request"},
		{"trailing json", nil, valid + `{}`, 400, "invalid_request"},
		{"over limit", nil, valid + strings.Repeat(" ", maxRequestBytes), 413, "request_too_large"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", tc.body)
			if tc.mutate != nil {
				tc.mutate(request)
			}
			response := httptest.NewRecorder()
			newTestHandler(t, &actionsStub{enroll: attendance.EnrollResult{Status: attendance.EnrollCreated}}).ServeHTTP(response, request)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), `"code":"`+tc.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
			assertSafeResponse(t, response)
		})
	}

	for _, path := range []string{"/miniapp/api/v1/attendance/enrollments", "/miniapp/api/v1/attendance/options", "/miniapp/api/v1/attendance/punches"} {
		request := apiRequest(http.MethodGet, path, "")
		if path == "/miniapp/api/v1/attendance/options" {
			request.Method = http.MethodPost
		}
		response := httptest.NewRecorder()
		newTestHandler(t, &actionsStub{}).ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Errorf("wrong method %s=%d", path, response.Code)
		}
	}
}

func TestEveryAttendanceRouteRequiresOriginAndBearer(t *testing.T) {
	routes := []struct{ method, path, body string }{
		{http.MethodPost, "/miniapp/api/v1/attendance/enrollments", `{"enrollmentCode":"code"}`},
		{http.MethodGet, "/miniapp/api/v1/attendance/options", ""},
		{http.MethodPost, "/miniapp/api/v1/attendance/punches", `{"shiftInstanceId":"shift","kind":"clock_in","latitude":25,"longitude":121,"accuracyMeters":5}`},
	}
	for _, route := range routes {
		for _, header := range []string{"Origin", "Authorization"} {
			request := apiRequest(route.method, route.path, route.body)
			request.Header.Del(header)
			response := httptest.NewRecorder()
			newTestHandler(t, &actionsStub{}).ServeHTTP(response, request)
			want := http.StatusForbidden
			if header == "Authorization" {
				want = http.StatusUnauthorized
			} else if route.method == http.MethodGet {
				want = http.StatusOK
			}
			if response.Code != want {
				t.Errorf("%s without %s=%d, want %d", route.path, header, response.Code, want)
			}
		}
		request := apiRequest(route.method, route.path, route.body)
		request.Header.Del("Sec-Fetch-Site")
		response := httptest.NewRecorder()
		newTestHandler(t, &actionsStub{}).ServeHTTP(response, request)
		if response.Code == http.StatusForbidden {
			t.Errorf("%s rejected absent optional Fetch Metadata", route.path)
		}
	}
}

func TestOptionsWithoutOriginRequiresSameOriginFetchMetadata(t *testing.T) {
	request := apiRequest(http.MethodGet, "/miniapp/api/v1/attendance/options", "")
	request.Header.Del("Origin")
	request.Header.Del("Sec-Fetch-Site")
	response := httptest.NewRecorder()
	newTestHandler(t, &actionsStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("options without browser origin signals=%d", response.Code)
	}
}

func TestApplicationErrorsUseSafeStableEnvelope(t *testing.T) {
	secret := errors.New("provider secret")
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{attendance.ErrInvalidInput, 400, "invalid_request"}, {attendance.ErrUnauthenticated, 401, "unauthorized"},
		{attendance.ErrEnrollmentInvalid, 404, "enrollment_invalid"}, {attendance.ErrNotAssigned, 403, "not_assigned"},
		{attendance.ErrSubmissionClosed, 409, "submission_closed"}, {attendance.ErrConflict, 409, "attendance_conflict"},
		{attendance.ErrRepositoryUnavailable, 503, "attendance_unavailable"}, {secret, 500, "internal_error"},
	}
	for _, tc := range tests {
		response := httptest.NewRecorder()
		newTestHandler(t, &actionsStub{err: tc.err}).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", `{"enrollmentCode":"code"}`))
		if response.Code != tc.status || !strings.Contains(response.Body.String(), `"code":"`+tc.code+`"`) || strings.Contains(response.Body.String(), secret.Error()) {
			t.Errorf("error %v response=%d %s", tc.err, response.Code, response.Body.String())
		}
	}
}

func TestLegacyEndpointIsRemoved(t *testing.T) {
	response := httptest.NewRecorder()
	legacyPath := "/miniapp/api/v1/" + "location-" + "validations"
	newTestHandler(t, &actionsStub{}).ServeHTTP(response, apiRequest(http.MethodPost, legacyPath, `{}`))
	if response.Code != http.StatusNotFound {
		t.Fatalf("old route status=%d", response.Code)
	}
}

func apiRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, testOrigin+path, strings.NewReader(body))
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Authorization", "Bearer id-token")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func assertSafeResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	for _, name := range []string{"Cache-Control", "Content-Security-Policy", "Permissions-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options"} {
		if response.Header().Get(name) == "" {
			t.Errorf("missing security header %s", name)
		}
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS unexpectedly enabled")
	}
	var payload any
	if strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") && json.Unmarshal(response.Body.Bytes(), &payload) != nil {
		t.Fatal("invalid JSON response")
	}
}
