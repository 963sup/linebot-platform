package attendance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/963sup/linebot-platform/internal/domain/attendance/aggregates"
	"github.com/963sup/linebot-platform/internal/domain/attendance/entities"
	"github.com/963sup/linebot-platform/internal/domain/attendance/repositories"
	valueobjects "github.com/963sup/linebot-platform/internal/domain/attendance/value-objects"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type countingClock struct {
	now   time.Time
	calls int
}

func (c *countingClock) Now() time.Time {
	c.calls++
	return c.now
}

type verifierFake struct {
	value VerifiedPrincipal
	err   error
}

func (f verifierFake) Verify(context.Context, string) (VerifiedPrincipal, error) {
	return f.value, f.err
}

type enrollmentFake struct {
	participant entities.Participant
	outcome     valueobjects.EnrollmentOutcome
	err         error
	hash        valueobjects.EnrollmentCodeHash
	inviteCalls int
	consumedAt  time.Time
}

func (f *enrollmentFake) Invite(context.Context, entities.Participant, entities.EnrollmentInvitation) error {
	f.inviteCalls++
	return f.err
}
func (f *enrollmentFake) ConsumeEnrollment(_ context.Context, h valueobjects.EnrollmentCodeHash, _ valueobjects.ParticipantID, at time.Time) (entities.Participant, valueobjects.EnrollmentOutcome, error) {
	f.hash = h
	f.consumedAt = at
	return f.participant, f.outcome, f.err
}

type currentAttendanceReaderFake struct {
	resolved     CurrentAttendanceSnapshot
	options      []CurrentAttendanceSnapshot
	attendance   *attendanceRepoFake
	err          error
	resolveCalls int
	listCalls    int
	resolveAt    time.Time
	listAt       time.Time
}

func (f *currentAttendanceReaderFake) ResolvePunchContext(_ context.Context, _ valueobjects.ParticipantID, _ valueobjects.ShiftInstanceID, at time.Time) (CurrentAttendanceSnapshot, error) {
	f.resolveCalls++
	f.resolveAt = at
	return f.withCommittedAttendance(f.resolved), f.err
}

func (f *currentAttendanceReaderFake) ListCurrentOptions(_ context.Context, _ valueobjects.ParticipantID, at time.Time) ([]CurrentAttendanceSnapshot, error) {
	f.listCalls++
	f.listAt = at
	if f.err != nil {
		return nil, f.err
	}
	if f.options != nil {
		return f.options, nil
	}
	return []CurrentAttendanceSnapshot{f.withCommittedAttendance(f.resolved)}, nil
}

func (f *currentAttendanceReaderFake) withCommittedAttendance(snapshot CurrentAttendanceSnapshot) CurrentAttendanceSnapshot {
	if f.attendance != nil && f.attendance.findErr == nil {
		value := f.attendance.value
		snapshot.Attendance = &value
	}
	return snapshot
}

type participantRepoFake struct {
	participant entities.Participant
	err         error
}

func (f *participantRepoFake) FindByEmployeeID(context.Context, valueobjects.EmployeeID) (entities.Participant, error) {
	return f.participant, f.err
}
func (f *participantRepoFake) FindByParticipantID(context.Context, valueobjects.ParticipantID) (entities.Participant, error) {
	return f.participant, f.err
}
func (f *participantRepoFake) SaveParticipant(context.Context, entities.Participant) error {
	return f.err
}

type workplaceRepoFake struct {
	value entities.Workplace
	err   error
}

func (f *workplaceRepoFake) FindWorkplace(context.Context, valueobjects.WorkplaceID) (entities.Workplace, error) {
	return f.value, f.err
}
func (f *workplaceRepoFake) SaveWorkplace(context.Context, entities.Workplace) error { return f.err }

type shiftRepoFake struct {
	shift           entities.ShiftInstance
	savedAssignment entities.ShiftAssignment
	err             error
	saved           bool
}

func (f *shiftRepoFake) FindShift(context.Context, valueobjects.ShiftInstanceID) (entities.ShiftInstance, error) {
	return f.shift, f.err
}
func (f *shiftRepoFake) SaveShift(context.Context, entities.ShiftInstance) error {
	f.saved = true
	return f.err
}
func (f *shiftRepoFake) SaveAssignment(_ context.Context, assignment entities.ShiftAssignment) error {
	f.savedAssignment = assignment
	return f.err
}

type attendanceRepoFake struct {
	value            aggregates.Attendance
	findErr, saveErr error
	commitOnError    bool
	saved            bool
	expiry           time.Time
	evidence         valueobjects.LocationEvidence
}

func (f *attendanceRepoFake) Find(context.Context, valueobjects.ParticipantID, valueobjects.ShiftAssignmentID) (aggregates.Attendance, error) {
	return f.value, f.findErr
}
func (f *attendanceRepoFake) ListReviewRequired(context.Context, int) ([]aggregates.Attendance, error) {
	return nil, f.findErr
}
func (f *attendanceRepoFake) SavePunch(_ context.Context, v aggregates.Attendance, _ uint64, _ valueobjects.PunchID, e valueobjects.LocationEvidence, expiry time.Time) error {
	f.saved = true
	f.value = v
	if f.saveErr == nil || f.commitOnError {
		f.findErr = nil
	}
	f.evidence = e
	f.expiry = expiry
	return f.saveErr
}
func (f *attendanceRepoFake) SaveReview(context.Context, aggregates.Attendance, uint64) error {
	return f.saveErr
}

func attendanceFixture(tb testing.TB) (Dependencies, valueobjects.ParticipantID) {
	tb.Helper()
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	pid, _ := valueobjects.NewParticipantID("participant")
	eid, _ := valueobjects.NewEmployeeID("employee")
	p, _ := entities.NewParticipant(eid, "E1", "Employee")
	_, _ = p.Bind(pid)
	wid, _ := valueobjects.NewWorkplaceID("workplace")
	point, _ := valueobjects.NewGeoPoint(25, 121)
	w, _ := entities.DefineWorkplace(wid, "Workplace", point, 100, 50, "policy-v1")
	sid, _ := valueobjects.NewShiftInstanceID("shift")
	window, _ := valueobjects.NewTimeWindow(now.Add(-time.Hour), now.Add(time.Hour))
	shift, _ := entities.DefineShiftInstance(sid, wid, "Shift", window, window, now.Add(-2*time.Hour), now.Add(2*time.Hour), "policy-v1")
	aid, _ := valueobjects.NewShiftAssignmentID("assignment")
	assignment, _ := entities.NewShiftAssignment(aid, eid, sid)
	enrollment := &enrollmentFake{participant: p}
	attendance := &attendanceRepoFake{findErr: repositories.ErrAttendanceNotFound}
	reader := &currentAttendanceReaderFake{
		resolved:   CurrentAttendanceSnapshot{Participant: p, Assignment: assignment, Shift: shift, Workplace: w},
		attendance: attendance,
	}
	return Dependencies{Verifier: verifierFake{value: VerifiedPrincipal{ParticipantID: pid, ValidatedAt: now}}, Enrollment: enrollment, Reader: reader, Attendance: attendance, Clock: fixedClock{now}}, pid
}

func currentOptionSnapshots(tb testing.TB, participant entities.Participant, count int, now time.Time) []CurrentAttendanceSnapshot {
	tb.Helper()
	values := make([]CurrentAttendanceSnapshot, 0, count)
	for i := 0; i < count; i++ {
		workplaceID, _ := valueobjects.NewWorkplaceID(fmt.Sprintf("workplace-%d", i))
		point, _ := valueobjects.NewGeoPoint(25, 121)
		workplace, err := entities.DefineWorkplace(workplaceID, fmt.Sprintf("Workplace %d", i), point, 100, 50, "policy-v1")
		if err != nil {
			tb.Fatal(err)
		}
		shiftID, _ := valueobjects.NewShiftInstanceID(fmt.Sprintf("shift-%d", i))
		window, windowErr := valueobjects.NewTimeWindow(now.Add(-time.Hour), now.Add(time.Hour))
		if windowErr != nil {
			tb.Fatal(windowErr)
		}
		shift, err := entities.DefineShiftInstance(shiftID, workplaceID, fmt.Sprintf("Shift %d", i), window, window, now.Add(-2*time.Hour), now.Add(2*time.Hour), "policy-v1")
		if err != nil {
			tb.Fatal(err)
		}
		assignmentID, _ := valueobjects.NewShiftAssignmentID(fmt.Sprintf("assignment-%d", i))
		assignment, err := entities.NewShiftAssignment(assignmentID, participant.EmployeeID(), shiftID)
		if err != nil {
			tb.Fatal(err)
		}
		values = append(values, CurrentAttendanceSnapshot{Participant: participant, Assignment: assignment, Shift: shift, Workplace: workplace})
	}
	return values
}

func TestServiceEnrollHashesCodeAndReturnsNoIdentity(t *testing.T) {
	deps, _ := attendanceFixture(t)
	deps.Enrollment.(*enrollmentFake).outcome = valueobjects.EnrollmentBound
	service, _ := NewService(deps)
	result, err := service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: "one-time"})
	if err != nil || result.Status != EnrollCreated {
		t.Fatalf("Enroll=(%+v,%v)", result, err)
	}
	got := deps.Enrollment.(*enrollmentFake).hash
	if got != enrollmentCodeHash("one-time") || string(got) == "one-time" {
		t.Fatal("enrollment code was not one-way hashed")
	}
	deps.Enrollment.(*enrollmentFake).outcome = valueobjects.EnrollmentAlreadyBound
	result, err = service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: "one-time"})
	if err != nil || result.Status != EnrollAlreadyCompleted {
		t.Fatalf("idempotent Enroll=(%+v,%v)", result, err)
	}
}

func TestServiceEnrollEnforcesOpenAPICodeLength(t *testing.T) {
	deps, _ := attendanceFixture(t)
	deps.Enrollment.(*enrollmentFake).outcome = valueobjects.EnrollmentBound
	service, _ := NewService(deps)
	valid := strings.Repeat("碼", maxEnrollmentCodeLength)
	if _, err := service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: valid}); err != nil {
		t.Fatalf("accepted OpenAPI maxLength boundary returned %v", err)
	}
	invalid := valid + "碼"
	if _, err := service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: invalid}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("overlong enrollment code error=%v", err)
	}
}

func TestServiceEnrollRejectsUnknownRepositoryOutcome(t *testing.T) {
	deps, _ := attendanceFixture(t)
	deps.Enrollment.(*enrollmentFake).outcome = valueobjects.EnrollmentOutcome("future-outcome")
	service, _ := NewService(deps)

	if _, err := service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: "one-time"}); !errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("unknown enrollment outcome error=%v", err)
	}
}

func TestServicePunchCreatesFirstAttendanceAndStoresRawEvidenceAtomically(t *testing.T) {
	deps, _ := attendanceFixture(t)
	service, _ := NewService(deps)
	result, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn, Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5}})
	if err != nil {
		t.Fatal(err)
	}
	repo := deps.Attendance.(*attendanceRepoFake)
	if !repo.saved || result.Status != PunchRecorded || result.PunchID == "" || result.AttendanceID != "assignment" {
		t.Fatalf("result=%+v saved=%v", result, repo.saved)
	}
	if !repo.expiry.Equal(deps.Clock.Now().Add(90 * 24 * time.Hour)) {
		t.Fatalf("expiry=%v", repo.expiry)
	}
	if repo.evidence.Latitude() != 25 || repo.evidence.AccuracyMeters() != 5 {
		t.Fatal("raw evidence not passed atomically")
	}
}

func TestServiceOptionsExposeOnlyNextValidPunch(t *testing.T) {
	deps, _ := attendanceFixture(t)
	service, _ := NewService(deps)
	before, err := service.ListOptions(context.Background(), OptionsCommand{IDToken: "token"})
	if err != nil || len(before.Options) != 1 || len(before.Options[0].AllowedKinds) != 1 || before.Options[0].AllowedKinds[0] != ClockIn {
		t.Fatalf("before clock-in options=%+v err=%v", before, err)
	}
	if _, err := service.Punch(context.Background(), PunchCommand{
		IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn,
		Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5},
	}); err != nil {
		t.Fatal(err)
	}
	after, err := service.ListOptions(context.Background(), OptionsCommand{IDToken: "token"})
	if err != nil || len(after.Options) != 1 || len(after.Options[0].AllowedKinds) != 1 || after.Options[0].AllowedKinds[0] != ClockOut {
		t.Fatalf("after clock-in options=%+v err=%v", after, err)
	}
}

func TestServiceListOptionsUsesOneCurrentReaderCall(t *testing.T) {
	for _, count := range []int{1, 20, 100} {
		t.Run(fmt.Sprintf("N=%d", count), func(t *testing.T) {
			deps, _ := attendanceFixture(t)
			reader := deps.Reader.(*currentAttendanceReaderFake)
			reader.options = currentOptionSnapshots(t, reader.resolved.Participant, count, deps.Clock.Now())
			clock := &countingClock{now: deps.Clock.Now()}
			deps.Clock = clock
			service, _ := NewService(deps)

			result, err := service.ListOptions(context.Background(), OptionsCommand{IDToken: "token"})
			if err != nil || len(result.Options) != count {
				t.Fatalf("ListOptions count=%d, err=%v", len(result.Options), err)
			}
			if reader.listCalls != 1 || reader.resolveCalls != 0 {
				t.Fatalf("reader calls list=%d resolve=%d", reader.listCalls, reader.resolveCalls)
			}
			if clock.calls != 1 || !reader.listAt.Equal(clock.now) {
				t.Fatalf("clock calls=%d, reader at=%v, want %v", clock.calls, reader.listAt, clock.now)
			}
		})
	}
}

func TestServiceListOptionsRejectsReaderErrorAndDanglingSnapshot(t *testing.T) {
	t.Run("reader error", func(t *testing.T) {
		deps, _ := attendanceFixture(t)
		reader := deps.Reader.(*currentAttendanceReaderFake)
		reader.err = repositories.ErrNotFound
		service, _ := NewService(deps)
		if _, err := service.ListOptions(context.Background(), OptionsCommand{IDToken: "token"}); !errors.Is(err, ErrRepositoryUnavailable) {
			t.Fatalf("ListOptions error=%v", err)
		}
		if reader.listCalls != 1 {
			t.Fatalf("reader calls=%d", reader.listCalls)
		}
	})

	t.Run("dangling assignment", func(t *testing.T) {
		deps, _ := attendanceFixture(t)
		reader := deps.Reader.(*currentAttendanceReaderFake)
		otherShiftID, _ := valueobjects.NewShiftInstanceID("other-shift")
		assignment, _ := entities.NewShiftAssignment(reader.resolved.Assignment.ID(), reader.resolved.Participant.EmployeeID(), otherShiftID)
		reader.resolved.Assignment = assignment
		service, _ := NewService(deps)
		if _, err := service.ListOptions(context.Background(), OptionsCommand{IDToken: "token"}); !errors.Is(err, ErrRepositoryUnavailable) {
			t.Fatalf("ListOptions error=%v", err)
		}
		if reader.listCalls != 1 {
			t.Fatalf("reader calls=%d", reader.listCalls)
		}
	})
}

func TestServicePunchRejectsInactiveAndDanglingCurrentResources(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *CurrentAttendanceSnapshot, time.Time)
		want   error
	}{
		{
			name: "inactive participant",
			mutate: func(t *testing.T, snapshot *CurrentAttendanceSnapshot, now time.Time) {
				t.Helper()
				if err := snapshot.Participant.Deactivate(now); err != nil {
					t.Fatal(err)
				}
			},
			want: ErrNotEnrolled,
		},
		{
			name: "inactive shift",
			mutate: func(t *testing.T, snapshot *CurrentAttendanceSnapshot, _ time.Time) {
				t.Helper()
				if err := snapshot.Shift.Deactivate(); err != nil {
					t.Fatal(err)
				}
			},
			want: ErrNotAssigned,
		},
		{
			name: "inactive workplace",
			mutate: func(t *testing.T, snapshot *CurrentAttendanceSnapshot, _ time.Time) {
				t.Helper()
				if err := snapshot.Workplace.Deactivate(); err != nil {
					t.Fatal(err)
				}
			},
			want: ErrNotAssigned,
		},
		{
			name: "dangling assignment",
			mutate: func(t *testing.T, snapshot *CurrentAttendanceSnapshot, _ time.Time) {
				t.Helper()
				otherShiftID, _ := valueobjects.NewShiftInstanceID("other-shift")
				assignment, err := entities.NewShiftAssignment(snapshot.Assignment.ID(), snapshot.Participant.EmployeeID(), otherShiftID)
				if err != nil {
					t.Fatal(err)
				}
				snapshot.Assignment = assignment
			},
			want: ErrRepositoryUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps, _ := attendanceFixture(t)
			reader := deps.Reader.(*currentAttendanceReaderFake)
			test.mutate(t, &reader.resolved, deps.Clock.Now())
			service, _ := NewService(deps)

			_, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn, Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5}})
			if !errors.Is(err, test.want) {
				t.Fatalf("Punch error=%v, want %v", err, test.want)
			}
			if reader.resolveCalls != 1 || reader.listCalls != 0 {
				t.Fatalf("reader calls resolve=%d list=%d", reader.resolveCalls, reader.listCalls)
			}
			if deps.Attendance.(*attendanceRepoFake).saved {
				t.Fatal("saved punch for rejected snapshot")
			}
		})
	}
}

func TestServiceCommandsUseOneServerTime(t *testing.T) {
	t.Run("enroll", func(t *testing.T) {
		deps, _ := attendanceFixture(t)
		clock := &countingClock{now: deps.Clock.Now()}
		deps.Clock = clock
		enrollment := deps.Enrollment.(*enrollmentFake)
		enrollment.outcome = valueobjects.EnrollmentBound
		service, _ := NewService(deps)
		if _, err := service.Enroll(context.Background(), EnrollCommand{IDToken: "token", Code: "code"}); err != nil {
			t.Fatal(err)
		}
		if clock.calls != 1 || !enrollment.consumedAt.Equal(clock.now) {
			t.Fatalf("clock calls=%d consumedAt=%v, want %v", clock.calls, enrollment.consumedAt, clock.now)
		}
	})

	t.Run("punch", func(t *testing.T) {
		deps, _ := attendanceFixture(t)
		clock := &countingClock{now: deps.Clock.Now()}
		deps.Clock = clock
		reader := deps.Reader.(*currentAttendanceReaderFake)
		service, _ := NewService(deps)
		result, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn, Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5}})
		if err != nil {
			t.Fatal(err)
		}
		repo := deps.Attendance.(*attendanceRepoFake)
		if clock.calls != 1 || reader.resolveCalls != 1 || !reader.resolveAt.Equal(clock.now) || !result.RecordedAt.Equal(clock.now) || !repo.expiry.Equal(clock.now.Add(rawEvidenceRetention)) {
			t.Fatalf("clock calls=%d reader calls=%d readerAt=%v recordedAt=%v expiry=%v", clock.calls, reader.resolveCalls, reader.resolveAt, result.RecordedAt, repo.expiry)
		}
	})
}

func TestServicePunchMapsOrderingConflictAndReplaysAfterWindow(t *testing.T) {
	deps, _ := attendanceFixture(t)
	service, _ := NewService(deps)
	command := PunchCommand{
		IDToken: "token", ShiftInstanceID: "shift", Kind: ClockOut,
		Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5},
	}
	if _, err := service.Punch(context.Background(), command); !errors.Is(err, ErrConflict) {
		t.Fatalf("clock-out-before-in error=%v", err)
	}
	command.Kind = ClockIn
	first, err := service.Punch(context.Background(), command)
	if err != nil || first.Status != PunchRecorded {
		t.Fatalf("first punch=(%+v,%v)", first, err)
	}
	deps.Clock = fixedClock{deps.Clock.Now().Add(3 * time.Hour)}
	service, _ = NewService(deps)
	replay, err := service.Punch(context.Background(), command)
	if err != nil || replay.Status != PunchAlreadyRecorded || replay.PunchID != first.PunchID {
		t.Fatalf("late replay=(%+v,%v), first=%+v", replay, err, first)
	}
}

func TestServiceConcurrentFirstPunchConvergesToCommittedResult(t *testing.T) {
	deps, _ := attendanceFixture(t)
	repo := deps.Attendance.(*attendanceRepoFake)
	repo.saveErr = repositories.ErrConflict
	repo.commitOnError = true
	service, _ := NewService(deps)
	result, err := service.Punch(context.Background(), PunchCommand{
		IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn,
		Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5},
	})
	if err != nil || result.Status != PunchAlreadyRecorded || result.PunchID == "" {
		t.Fatalf("concurrent convergence=(%+v,%v)", result, err)
	}
}

func TestServicePunchRejectsInvalidAccuracyAndMapsRepositoryError(t *testing.T) {
	deps, _ := attendanceFixture(t)
	service, _ := NewService(deps)
	if _, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn, Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 0}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("accuracy error=%v", err)
	}
	sentinel := errors.New("repository outage")
	deps.Reader.(*currentAttendanceReaderFake).err = sentinel
	service, _ = NewService(deps)
	if _, err := service.Punch(context.Background(), PunchCommand{IDToken: "token", ShiftInstanceID: "shift", Kind: ClockIn, Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5}}); !errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("repository error=%v", err)
	}
}

func TestServicePunchPreservesAuthenticationError(t *testing.T) {
	deps, _ := attendanceFixture(t)
	deps.Verifier = verifierFake{err: ErrUnauthenticated}
	service, _ := NewService(deps)
	if _, err := service.Punch(context.Background(), PunchCommand{IDToken: "invalid", ShiftInstanceID: "shift", Kind: ClockIn, Location: LocationFix{Latitude: 25, Longitude: 121, AccuracyMeters: 5}}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("authentication error=%v", err)
	}
}

func TestServicePreservesCancellation(t *testing.T) {
	deps, _ := attendanceFixture(t)
	service, _ := NewService(deps)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ListOptions(ctx, OptionsCommand{IDToken: "token"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func BenchmarkServiceListOptions100(b *testing.B) {
	deps, _ := attendanceFixture(b)
	reader := deps.Reader.(*currentAttendanceReaderFake)
	reader.options = currentOptionSnapshots(b, reader.resolved.Participant, 100, deps.Clock.Now())
	service, err := NewService(deps)
	if err != nil {
		b.Fatal(err)
	}
	command := OptionsCommand{IDToken: "token"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := service.ListOptions(context.Background(), command); err != nil {
			b.Fatal(err)
		}
	}
}
