package architecture

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestMiniAppOpenAPIContract(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi/miniapp.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	text := string(data)
	for _, marker := range []string{
		"  /miniapp:\n", "/miniapp/:", "/miniapp/assets/app.css:", "/miniapp/assets/app.js:",
		"/miniapp/config.json:", "/miniapp/api/v1/attendance/enrollments:",
		"/miniapp/api/v1/attendance/today:", "/miniapp/api/v1/attendance/punches:",
		"liff.getIDToken()", "clock_in", "clock_out", "allowedKinds", "text/javascript:",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("OpenAPI missing Attendance contract marker %q", marker)
		}
	}
	for _, forbidden := range []string{"/location-validations", "/attendance/options", "shiftInstanceId", "workplaceLabel", "reviewState", "attendanceId", "punchId", "Bearer <LIFF access token>", "subject:"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("OpenAPI retains forbidden contract %q", forbidden)
		}
	}

	paths := requireStringMap(t, document["paths"], "paths")
	assertResponses(t, paths, "/miniapp/api/v1/attendance/enrollments", "post", "200", "201", "400", "401", "403", "404", "405", "409", "413", "415", "500", "503")
	assertResponses(t, paths, "/miniapp/api/v1/attendance/today", "get", "200", "401", "403", "405", "409", "500", "503")
	assertResponses(t, paths, "/miniapp/api/v1/attendance/punches", "post", "200", "201", "400", "401", "403", "404", "405", "409", "413", "415", "500", "503")

	components := requireStringMap(t, document["components"], "components")
	schemas := requireStringMap(t, components["schemas"], "schemas")
	result := requireStringMap(t, schemas["PunchResult"], "PunchResult")
	properties := requireStringMap(t, result["properties"], "PunchResult properties")
	for _, forbidden := range []string{"latitude", "longitude", "accuracyMeters", "attendanceDayId", "punchId"} {
		if _, ok := properties[forbidden]; ok {
			t.Errorf("PunchResult discloses %s", forbidden)
		}
	}
}

func assertResponses(t *testing.T, paths map[string]any, path, method string, statuses ...string) {
	t.Helper()
	operation := requireStringMap(t, requireStringMap(t, paths[path], path)[method], path+" "+method)
	responses := requireStringMap(t, operation["responses"], path+" responses")
	for _, status := range statuses {
		if _, ok := responses[status]; !ok {
			t.Errorf("%s %s is missing response status %s", method, path, status)
		}
	}
}

func requireStringMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI %s is %T, want mapping", name, value)
	}
	return result
}
