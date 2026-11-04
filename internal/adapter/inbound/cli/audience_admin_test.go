package cli

import (
	"bytes"
	"testing"
)

func TestAudienceAdminFailsClosed(t *testing.T) {
	getenv:=func(string)string{return ""}
	var out bytes.Buffer
	for _,args:=range [][]string{
		{"--action","delete","--audience-id","1"},
		{"--action","delete","--audience-id","1","--apply","--confirm","create"},
		{"--action","create","--apply","--confirm","create"},
		{"--action","append-file","--audience-id","1","--apply","--confirm","append-file"},
	} {
		if err:=runAudienceAdmin(args,getenv,&out);err==nil{t.Fatalf("accepted invalid %v",args)}
	}
	if out.Len()!=0{t.Fatal("invalid action leaked data")}
}
