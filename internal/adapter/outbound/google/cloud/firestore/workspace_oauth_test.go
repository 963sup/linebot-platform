package firestore

import (
	"strings"
	"testing"
)

func TestOAuthDocumentKeysAreOpaqueAndScoped(t *testing.T) {
	id, err := connectionID("line-user-1","google-subject-1")
	if err != nil { t.Fatal(err) }
	id2, err := connectionID("line-user-2","google-subject-1")
	if err != nil { t.Fatal(err) }
	if id == id2 || len(id) != 64 || strings.Contains(id,"line-user") {
		t.Fatal("OAuth connection document leaked or mixed identities")
	}
	if _, err := connectionID("","subject"); err == nil { t.Fatal("empty owner accepted") }
	if _, err := connectionID("line",""); err == nil { t.Fatal("empty subject accepted") }
	state := strings.Repeat("A",43)
	if _, err := opaqueStateID(state); err != nil { t.Fatal(err) }
	if _, err := opaqueStateID("not-state"); err == nil { t.Fatal("invalid state accepted") }
}
