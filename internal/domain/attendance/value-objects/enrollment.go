package valueobjects

type EnrollmentOutcome string

const (
	EnrollmentBound        EnrollmentOutcome = "bound"
	EnrollmentAlreadyBound EnrollmentOutcome = "already_bound"
)

func (o EnrollmentOutcome) Valid() bool {
	return o == EnrollmentBound || o == EnrollmentAlreadyBound
}
