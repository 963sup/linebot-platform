package valueobjects

type ConnectionStatus string

const (
	ConnectionActive       ConnectionStatus = "active"
	ConnectionDisconnected ConnectionStatus = "disconnected"
)

func (s ConnectionStatus) Valid() bool {
	return s == ConnectionActive || s == ConnectionDisconnected
}
