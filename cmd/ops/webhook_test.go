package main
import("bytes";"testing")
func TestWebhookAdminRequiresExplicitApproval(t *testing.T){
	env:=func(string)string{return ""}
	for _,args:=range [][]string{
		{"--action","set","--endpoint","https://example.com"},
		{"--action","set","--endpoint","https://example.com","--apply","--confirm","test"},
		{"--action","test","--endpoint","http://example.com","--apply","--confirm","test"},
	}{
		if err:=runWebhookAdmin(args,env,&bytes.Buffer{});err==nil{t.Fatalf("accepted invalid admin args %v",args)}
	}
}
