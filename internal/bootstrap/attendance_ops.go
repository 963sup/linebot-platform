package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/adapter/inbound/cli"
	firestoreout "github.com/963sup/linebot-platform/internal/adapter/outbound/google/cloud/firestore"
	attendanceapp "github.com/963sup/linebot-platform/internal/application/usecases/attendance"
	firestoreinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/firestore"
	secretmanagerinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/secretmanager"
)

func newAttendanceOpsRuntime(ctx context.Context, config cli.AttendanceRuntimeConfig) (cli.AttendanceRuntime, error) {
	if ctx == nil {
		return nil, errors.New("Attendance Ops context is required")
	}
	if config.Environment != string(EnvLocal) && config.Environment != string(EnvStaging) && config.Environment != string(EnvProduction) {
		return nil, errors.New("Attendance Ops environment is invalid")
	}
	if !projectIDPattern.MatchString(config.ProjectID) || config.ProjectID != strings.TrimSpace(config.ProjectID) {
		return nil, errors.New("Attendance Ops project ID is invalid")
	}
	versions, err := parseAttendanceKeyVersions(config.KeyVersions, config.ProjectID)
	if err != nil {
		return nil, err
	}
	if _, ok := versions[config.ActiveKeyID]; !ok {
		return nil, errors.New("Attendance Ops active key is missing")
	}
	secretClient, err := secretmanagerinfra.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	keyring, keyErr := secretmanagerinfra.LoadAttendanceKeyring(ctx, secretClient, config.ActiveKeyID, versions)
	closeSecretErr := secretClient.Close()
	if keyErr != nil || closeSecretErr != nil {
		return nil, errors.Join(keyErr, closeSecretErr)
	}
	database := config.FirestoreDatabase
	if database == "" {
		database = firestoreinfra.DefaultDatabase
	}
	client, err := firestoreinfra.NewClientWithDatabase(ctx, config.ProjectID, database)
	if err != nil {
		return nil, err
	}
	store, err := firestoreout.NewAttendanceStore(client, keyring)
	if err != nil {
		return nil, errors.Join(err, client.Close())
	}
	service, err := attendanceapp.NewAdminService(attendanceapp.AdminDependencies{
		Enrollment: store, Participants: store, Shifts: store,
		Workplaces: store, Attendance: store, Clock: opsClock{}, Evidence: store,
	}, secureEnrollmentTokenGenerator{})
	if err != nil {
		return nil, errors.Join(err, client.Close())
	}
	runtime, err := cli.NewAttendanceRuntime(service, client.Close)
	if err != nil {
		return nil, errors.Join(err, client.Close())
	}
	return runtime, nil
}

type opsClock struct{}

func (opsClock) Now() time.Time { return time.Now().UTC() }

type secureEnrollmentTokenGenerator struct{}

func (secureEnrollmentTokenGenerator) Generate() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", errors.New("generate Attendance enrollment token")
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
