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
			required: []string{"唯一 strategically implemented Bounded Context", "ADR 0003 授權的 `candidate/hypothesis` tactical models", "不得因已有程式碼、Adapter 或測試而宣稱 strategic promotion"},
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
			required: []string{"唯一 strategically implemented Bounded Context", "`candidate/hypothesis` tactical models"},
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
			required: []string{"目前只有 `internal/infrastructure/google/genai/` client factory", "尚未建立業務 outbound adapter"},
			forbid:   []string{"`internal/adapter/outbound/genai/`"},
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
