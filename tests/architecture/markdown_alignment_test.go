package architecture

import (
	"os"
	"strings"
	"testing"
)

func TestMarkdownArchitectureContractsStayAligned(t *testing.T) {
	tests := []struct {
		path     string
		required []string
		forbid   []string
	}{
		{
			path:     "../../SECURITY.md",
			required: []string{"開發用 MVP", "Staging／Production"},
			forbid:   []string{"僅為目錄骨架"},
		},
		{
			path:     "../../AGENTS.md",
			required: []string{"Application owns use-case orchestration and consumer-owned capability ports", "Domain may own repository contracts", "consuming Application capability or Domain repository"},
			forbid:   []string{"Application owns all outbound ports", "Domain must not own repository contracts", "Adapters only implement Application ports"},
		},
		{
			path:     "../../internal/application/AGENTS.md",
			required: []string{"工作流程與外部能力 Port 由唯一消費它的 capability 擁有", "Repository contract 可由 Domain 擁有", "不得在 Application 維護重複 contract"},
			forbid:   []string{"Repository Port 與事件分發應有明確失敗語意", "所有 Repository contract 都由 Application 擁有", "Domain 不得擁有 Repository contract"},
		},
		{
			path:     "../../internal/adapter/AGENTS.md",
			required: []string{"root parent context", "command deadline 必須從 parent 派生", "runtime 必須在命令結束時明確 `Close`", "Bootstrap 只組裝依賴與資源"},
			forbid:   []string{"Adapter 只實作 Application Port", "所有 Outbound Adapter 都只實作 Application Port"},
		},
		{
			path:     "../../internal/domain/AGENTS.md",
			required: []string{"`attendance/` 是 strategically implemented Bounded Context", "ADR 0003 授權的 `candidate/hypothesis` tactical models", "不得因已有程式碼、Adapter 或測試而宣稱 strategic promotion"},
		},
		{
			path:     "../../internal/infrastructure/AGENTS.md",
			required: []string{"限制各 client timeout、重試範圍、連線池與 shutdown", "顯式 Close", "不可對共享 Client 呼叫"},
		},
		{
			path:     "../../internal/adapter/README.md",
			required: []string{"Inbound adapters", "Application capability ports", "Domain repository contracts"},
			forbid:   []string{"translate provider APIs into application ports"},
		},
		{
			path:     "../../internal/domain/README.md",
			required: []string{"`attendance/`", "strategically implemented", "`candidate/hypothesis` tactical models"},
		},
		{
			path:     "../../docs/domain/context-map.md",
			required: []string{"## Relationships", "Decision", "Confidence", "hypothesis / medium"},
		},
		{
			path:     "../../docs/operations/staging-acceptance.md",
			required: []string{"17", "503", "Retry-After", "16", "C2 目前為 `BLOCKED`", "production image"},
		},
		{
			path:     "../../docs/versions.md",
			required: []string{"目前只有 `internal/infrastructure/google/genai/` client factory", "outbound adapter 尚未實作"},
			forbid:   []string{"`internal/adapter/outbound/genai/`"},
		},
		{
			path:     "../../docs/sources.md",
			required: []string{"目前只有 `internal/infrastructure/google/genai/` client factory", "尚未建立業務 outbound adapter", "`liff.getIDToken()`", "AES-GCM", "90 天 TTL"},
			forbid:   []string{"`internal/adapter/outbound/genai/`"},
		},
		{
			path:     "../../docs/line/miniapp-attendance.md",
			required: []string{"Attendance Domain owns", "shift instances", "`liff.getIDToken()`", "never return latitude", "Not Verified"},
			forbid:   []string{"LIFF access token>"},
		},
		{
			path:     "../../docs/architecture/decisions/0004-attendance-bounded-context.md",
			required: []string{"Attendance", "Implemented", "physical presence", "90 days", "five years"},
		},
		{
			path:     "../../docs/sdk/workspace-oauth.md",
			required: []string{"`internal/bootstrap/` 只有組裝與啟動責任"},
			forbid:   []string{"`bootstrap/` 只有組裝與啟動責任"},
		},
		{
			path:     "../../docs/operations/durable-worker.md",
			required: []string{"Capacity verification is currently `BLOCKED`", "containerConcurrency=16", "absent from the production image"},
		},
		{
			path:     "../../docs/architecture/decisions/0001-strategic-domain-model-authority.md",
			required: []string{"repository implementation evidence", "not deployed Production evidence"},
		},
	}

	for _, tt := range tests {
		data, err := os.ReadFile(tt.path)
		if err != nil {
			t.Errorf("read %s: %v", tt.path, err)
			continue
		}
		text := string(data)
		for _, marker := range tt.required {
			if !strings.Contains(text, marker) {
				t.Errorf("%s missing architecture marker %q", tt.path, marker)
			}
		}
		for _, marker := range tt.forbid {
			if strings.Contains(text, marker) {
				t.Errorf("%s retains obsolete architecture wording %q", tt.path, marker)
			}
		}
	}
}

func TestArchitectureSectionsPreserveOwnershipAndDirectionSemantics(t *testing.T) {
	tests := []struct {
		path      string
		heading   string
		required  []string
		forbidden []string
	}{
		{
			path:      "../../docs/architecture/blueprint.md",
			heading:   "### Ports",
			required:  []string{"Application 的 inbound use-case API 與 outbound capability port", "Domain repository contract", "兩者不得重複同一 contract"},
			forbidden: []string{"Application owns all outbound ports", "Domain must not own repository contracts"},
		},
		{
			path:      "../../docs/architecture/blueprint.md",
			heading:   "### Adapters",
			required:  []string{"Application capability", "Domain 擁有的 repository contract"},
			forbidden: []string{"Outbound adapters 實作 ports"},
		},
		{
			path:      "../../docs/architecture/blueprint.md",
			heading:   "## 2. 唯一依賴方向",
			required:  []string{"composition dependency", "Runtime 呼叫", "source dependency", "Application capability port 或 Domain repository contract"},
			forbidden: []string{"Runtime dependency direction"},
		},
		{
			path:      "../../docs/architecture/sdk-integration.md",
			heading:   "## Composition、runtime call 與 source dependency",
			required:  []string{"Composition ownership", "Runtime call", "Compile-time source dependency", "Application capability", "Domain repository contract"},
			forbidden: []string{"## Runtime dependency direction"},
		},
	}
	for _, test := range tests {
		t.Run(test.path+"/"+test.heading, func(t *testing.T) {
			data, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			section, ok := markdownSection(string(data), test.heading)
			if !ok {
				t.Fatalf("missing Markdown section %q", test.heading)
			}
			for _, marker := range test.required {
				if !strings.Contains(section, marker) {
					t.Errorf("section %q missing %q", test.heading, marker)
				}
			}
			for _, marker := range test.forbidden {
				if strings.Contains(section, marker) {
					t.Errorf("section %q contains obsolete wording %q", test.heading, marker)
				}
			}
		})
	}
}

func TestMarkdownSectionStopsAtPeerHeading(t *testing.T) {
	text := "## Target\nrequired\n### Child\nkept\n## Peer\nforbidden\n"
	section, ok := markdownSection(text, "## Target")
	if !ok || !strings.Contains(section, "kept") || strings.Contains(section, "forbidden") {
		t.Fatalf("unexpected section extraction: %q", section)
	}
}

func markdownSection(document, heading string) (string, bool) {
	lines := strings.Split(document, "\n")
	level := len(heading) - len(strings.TrimLeft(heading, "#"))
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = index
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(line, "#") {
			continue
		}
		nextLevel := len(line) - len(strings.TrimLeft(line, "#"))
		if nextLevel <= level && len(line) > nextLevel && line[nextLevel] == ' ' {
			end = index
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}
