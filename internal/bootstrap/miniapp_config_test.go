package bootstrap

import "testing"

func TestLoadMiniAppConfigDefaultsFirestoreDatabase(t *testing.T) {
	values := map[string]string{
		KeyAppEnv:             string(EnvLocal),
		KeyMiniAppChannelID:   "1234567890",
		KeyMiniAppLIFFURL:     "https://miniapp.line.me/1234567890-example",
		KeyMiniAppEndpointURL: "http://localhost:8080/miniapp/",
	}
	config, err := LoadMiniAppConfig(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if config.FirestoreDatabase != "(default)" {
		t.Fatalf("FirestoreDatabase = %q, want (default)", config.FirestoreDatabase)
	}
}

func TestLoadMiniAppConfigRejectsInvalidFirestoreDatabase(t *testing.T) {
	values := map[string]string{
		KeyAppEnv:             string(EnvLocal),
		KeyMiniAppChannelID:   "1234567890",
		KeyMiniAppLIFFURL:     "https://miniapp.line.me/1234567890-example",
		KeyMiniAppEndpointURL: "http://localhost:8080/miniapp/",
		KeyFirestoreDatabase:  "invalid/database",
	}
	if _, err := LoadMiniAppConfig(func(key string) string { return values[key] }); err == nil {
		t.Fatal("LoadMiniAppConfig accepted an invalid Firestore database ID")
	}
}
