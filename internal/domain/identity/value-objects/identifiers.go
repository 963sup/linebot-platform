// Package valueobjects defines validated immutable identity values.
package valueobjects

import (
 "errors"
 "strings"
)

type LineUserID struct{ value string }
type GoogleAccountID struct{ value string }

func NewLineUserID(value string) (LineUserID,error) {
 if value=="" || value!=strings.TrimSpace(value) { return LineUserID{},errors.New("invalid LINE user ID") }
 return LineUserID{value:value},nil
}
func NewGoogleAccountID(value string) (GoogleAccountID,error) {
 if value=="" || value!=strings.TrimSpace(value) { return GoogleAccountID{},errors.New("invalid Google account ID") }
 return GoogleAccountID{value:value},nil
}
func (id LineUserID) String() string { return id.value }
func (id GoogleAccountID) String() string { return id.value }
func (id LineUserID) Valid() bool { return id.value!="" }
func (id GoogleAccountID) Valid() bool { return id.value!="" }
