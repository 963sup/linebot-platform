package valueobjects

import "errors"

// DeliveryID is the stable, provider-private identity of one delivery.
type DeliveryID struct{ value string }

func ParseDeliveryID(value string) (DeliveryID, error) {
	if len(value) != 64 {
		return DeliveryID{}, errors.New("delivery ID must be 64 lowercase hexadecimal characters")
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return DeliveryID{}, errors.New("delivery ID must be 64 lowercase hexadecimal characters")
		}
	}
	return DeliveryID{value: value}, nil
}

func (id DeliveryID) String() string { return id.value }
func (id DeliveryID) Valid() bool {
	_, err := ParseDeliveryID(id.value)
	return err == nil
}
