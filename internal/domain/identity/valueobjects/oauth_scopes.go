package valueobjects

import (
 "errors"
 "sort"
 "strings"
)

// OAuthScopes is an immutable, canonical set of granted scopes.
type OAuthScopes struct{ values []string }
func NewOAuthScopes(scopes []string) (OAuthScopes,error) {
 if len(scopes)==0 { return OAuthScopes{},errors.New("OAuth scopes required") }
 unique:=make(map[string]struct{},len(scopes))
 for _,scope:=range scopes {
  if scope=="" || scope!=strings.TrimSpace(scope) || strings.ContainsAny(scope," \t\r\n") {
   return OAuthScopes{},errors.New("invalid OAuth scope")
  }
  unique[scope]=struct{}{}
 }
 values:=make([]string,0,len(unique))
 for scope:=range unique { values=append(values,scope) }
 sort.Strings(values)
 return OAuthScopes{values:values},nil
}
func (s OAuthScopes) Values() []string { return append([]string(nil),s.values...) }
func (s OAuthScopes) Contains(scope string) bool {
 idx:=sort.SearchStrings(s.values,scope)
 return idx<len(s.values) && s.values[idx]==scope
}
func (s OAuthScopes) Valid() bool { return len(s.values)>0 }
