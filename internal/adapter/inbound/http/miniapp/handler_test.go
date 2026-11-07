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
	token    string
	enroll   attendance.EnrollResult
	today    attendance.TodayResult
	punch    attendance.PunchResult
	err      error
	punchCmd attendance.PunchCommand
}

func (a *actionsStub) Enroll(_ context.Context, command attendance.EnrollCommand) (attendance.EnrollResult, error) {
	a.token = command.IDToken
	return a.enroll, a.err
}
func (a *actionsStub) GetToday(_ context.Context, query attendance.TodayQuery) (attendance.TodayResult, error) {
	a.token = query.IDToken
	return a.today, a.err
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
}

func TestAssetsExposeMinimalAttendanceJourney(t *testing.T) {
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
	for _, required := range []string{"attendance/enrollments", "attendance/today", "attendance/punches", "liff.getIDToken()", "getCurrentPosition", "enableHighAccuracy: false", "maximumAge: 0", "timeout: 10000", "clock_in", "clock_out", "content-type"} {
		if !strings.Contains(js.Body.String(), required) {
			t.Errorf("JavaScript missing %q", required)
		}
	}
	for _, forbidden := range []string{"attendance/options", "shiftInstanceId", "workplaceLabel", "localStorage", "sessionStorage", "watchPosition", "analytics"} {
		if strings.Contains(js.Body.String(), forbidden) {
			t.Errorf("JavaScript contains forbidden capability %q", forbidden)
		}
	}
}

func TestEnrollmentCreatedAndIdempotentStatuses(t *testing.T) {
	for _, tc := range []struct {
		value  attendance.EnrollStatus
		status int
	}{{attendance.EnrollCreated, 201}, {attendance.EnrollAlreadyCompleted, 200}} {
		actions := &actionsStub{enroll: attendance.EnrollResult{Status: tc.value}}
		response := httptest.NewRecorder()
		newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", `{"enrollmentCode":"invite-code"}`))
		if response.Code != tc.status || actions.token != "id-token" {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	}
}

func TestTodayReturnsEnrollmentAndDailyStateWithoutInternalIDs(t *testing.T) {
	response := httptest.NewRecorder()
	newTestHandler(t, &actionsStub{err: attendance.ErrNotEnrolled}).ServeHTTP(response, apiRequest(http.MethodGet, "/miniapp/api/v1/attendance/today", ""))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"enrollmentRequired":true`) {
		t.Fatalf("not enrolled=%d %s", response.Code, response.Body.String())
	}

	at := time.Date(2026, 11, 7, 1, 2, 3, 0, time.UTC)
	actions := &actionsStub{today: attendance.TodayResult{Workday: "2026-11-07", ClockIn: &attendance.PunchView{PunchID: "secret-punch", Kind: attendance.ClockIn, RecordedAt: at}, AllowedKinds: []attendance.PunchKind{attendance.ClockOut}}}
	response = httptest.NewRecorder()
	newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodGet, "/miniapp/api/v1/attendance/today", ""))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"workday":"2026-11-07"`) || !strings.Contains(response.Body.String(), `"allowedKinds":["clock_out"]`) || strings.Contains(response.Body.String(), "secret-punch") {
		t.Fatalf("today=%d %s", response.Code, response.Body.String())
	}
}

func TestPunchCreatedAndReplayNeverReturnCoordinatesOrInternalIDs(t *testing.T) {
	at := time.Date(2026, 11, 7, 1, 2, 3, 0, time.UTC)
	for _, tc := range []struct {
		status attendance.PunchStatus
		want   int
	}{{attendance.PunchRecorded, 201}, {attendance.PunchAlreadyRecorded, 200}} {
		actions := &actionsStub{punch: attendance.PunchResult{AttendanceDayID: "secret-day", Workday: "2026-11-07", Punch: attendance.PunchView{PunchID: "secret-punch", Kind: attendance.ClockIn, RecordedAt: at}, Status: tc.status}}
		response := httptest.NewRecorder()
		body := `{"kind":"clock_in","latitude":25.033,"longitude":121.565,"accuracyMeters":12.5}`
		newTestHandler(t, actions).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/punches", body))
		if response.Code != tc.want || !strings.Contains(response.Body.String(), `"workday":"2026-11-07"`) || actions.punchCmd.Location.AccuracyMeters != 12.5 {
			t.Fatalf("response=%d %s command=%+v", response.Code, response.Body.String(), actions.punchCmd)
		}
		for _, forbidden := range []string{"latitude", "longitude", "accuracyMeters", "secret-day", "secret-punch", "attendanceDayId", "punchId"} {
			if strings.Contains(response.Body.String(), forbidden) {
				t.Errorf("response leaked %q", forbidden)
			}
		}
	}
}

func TestPunchRejectsRemovedClientAuthority(t *testing.T) {
	for _, extra := range []string{`"shiftInstanceId":"shift",`, `"workplaceId":"workplace",`, `"employeeId":"employee",`, `"workday":"2026-11-07",`} {
		response := httptest.NewRecorder()
		body := `{` + extra + `"kind":"clock_in","latitude":25,"longitude":121,"accuracyMeters":5}`
		newTestHandler(t, &actionsStub{}).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/punches", body))
		if response.Code != http.StatusBadRequest {
			t.Errorf("accepted removed authority %s: %d", extra, response.Code)
		}
	}
}

func TestAttendanceAPISecurityAndMethods(t *testing.T) {
	valid := `{"enrollmentCode":"code"}`
	for _, tc := range []struct {
		name   string
		mutate func(*http.Request)
		status int
	}{{"missing origin", func(r *http.Request) { r.Header.Del("Origin") }, 403}, {"cross site", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, 403}, {"missing bearer", func(r *http.Request) { r.Header.Del("Authorization") }, 401}} {
		t.Run(tc.name, func(t *testing.T) {
			request := apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", valid)
			tc.mutate(request)
			response := httptest.NewRecorder()
			newTestHandler(t, &actionsStub{}).ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d", response.Code)
			}
		})
	}
	for _, route := range []struct{ method, path string }{{http.MethodGet, "/miniapp/api/v1/attendance/enrollments"}, {http.MethodPost, "/miniapp/api/v1/attendance/today"}, {http.MethodGet, "/miniapp/api/v1/attendance/punches"}} {
		response := httptest.NewRecorder()
		newTestHandler(t, &actionsStub{}).ServeHTTP(response, apiRequest(route.method, route.path, ""))
		if response.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s=%d", route.path, response.Code)
		}
	}
}

func TestApplicationErrorsUseSafeStableEnvelope(t *testing.T) {
	secret := errors.New("provider secret")
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{attendance.ErrInvalidInput, 400, "invalid_request"}, {attendance.ErrUnauthenticated, 401, "unauthorized"}, {attendance.ErrEnrollmentInvalid, 404, "enrollment_invalid"}, {attendance.ErrNotEnrolled, 403, "enrollment_required"}, {attendance.ErrConflict, 409, "attendance_conflict"}, {attendance.ErrRepositoryUnavailable, 503, "attendance_unavailable"}, {secret, 500, "internal_error"}} {
		response := httptest.NewRecorder()
		newTestHandler(t, &actionsStub{err: tc.err}).ServeHTTP(response, apiRequest(http.MethodPost, "/miniapp/api/v1/attendance/enrollments", `{"enrollmentCode":"code"}`))
		if response.Code != tc.status || !strings.Contains(response.Body.String(), `"code":"`+tc.code+`"`) || strings.Contains(response.Body.String(), secret.Error()) {
			t.Errorf("error %v response=%d %s", tc.err, response.Code, response.Body.String())
		}
	}
}

func TestRemovedOptionsEndpointIsNotFound(t *testing.T) {
	response := httptest.NewRecorder()
	newTestHandler(t, &actionsStub{}).ServeHTTP(response, apiRequest(http.MethodGet, "/miniapp/api/v1/attendance/options", ""))
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
