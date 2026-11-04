package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type workspaceMethod struct {
	Operation string `json:"operation"`
	Resource string `json:"resource"`
	Method string `json:"method"`
	HTTPMethod string `json:"httpMethod"`
	Path string `json:"path"`
	Mode string `json:"mode"`
	Status string `json:"status"`
	Scopes []string `json:"scopes"`
}

type workspaceInventory struct {
	SchemaVersion int `json:"schemaVersion"`
	API string `json:"api"`
	SDKModule string `json:"sdkModule"`
	SDKVersion string `json:"sdkVersion"`
	DiscoveryRevision string `json:"discoveryRevision"`
	Methods []workspaceMethod `json:"methods"`
}

func TestWorkspacePinnedEndpointInventory(t *testing.T) {
	for _, tc := range []struct{name string; want int}{
		{"calendar",38},
		{"tasks",14},
	} {
		t.Run(tc.name,func(t *testing.T){
			data, err:=os.ReadFile(filepath.Join("..","..","docs","google",tc.name+"-endpoints.json"))
			if err!=nil{t.Fatal(err)}
			var inventory workspaceInventory
			if err:=json.Unmarshal(data,&inventory);err!=nil{t.Fatal(err)}
			if inventory.SchemaVersion!=1 || inventory.API!=tc.name || inventory.SDKModule!="google.golang.org/api" || inventory.SDKVersion!="v0.310.0" || inventory.DiscoveryRevision=="" {t.Fatal("unsupported workspace inventory version")}
			if len(inventory.Methods)!=tc.want{t.Fatalf("got %d methods, want %d",len(inventory.Methods),tc.want)}
			seen:=map[string]bool{}
			for _,m:=range inventory.Methods{
				if m.Operation!=m.Resource+"."+m.Method || seen[m.Operation] || m.Path=="" || len(m.Scopes)==0 {t.Fatalf("invalid or duplicate method %q",m.Operation)}
				seen[m.Operation]=true
				if m.HTTPMethod=="GET" && m.Mode!="read"{t.Fatalf("GET operation %s must be read",m.Operation)}
				if m.HTTPMethod!="GET" && m.Mode!="write"{t.Fatalf("mutation %s must be write",m.Operation)}
				if m.Status!="planned" && m.Status!="integrated_offline" && m.Status!="staging_verified" {t.Fatalf("invalid status %q",m.Status)}
			}
		})
	}
}
