package architecture

import (
	"go/ast"
	"testing"
)

func TestBootstrapDoesNotOwnInboundPresentationMapping(t *testing.T) {
	forbiddenMethods := map[string]bool{
		"GetAudienceDefinition": true, "DefineAudience": true, "ReviseAudience": true,
		"ActivateAudience": true, "ArchiveAudience": true, "SyncAudience": true,
		"GetSurface": true, "DefineSurface": true, "ReviseSurface": true,
		"PublishSurface": true, "RetireSurface": true, "DeploySurface": true,
		"GetPromotion": true, "DefinePromotion": true, "ActivatePromotion": true,
		"ClosePromotion": true, "CreateDeliveryIntent": true,
		"GetAutomationRequest": true, "RequestAutomation": true,
		"ApproveAutomation": true, "ExecuteAutomation": true,
	}
	walkProductionGoFiles(t, "../../internal/bootstrap", func(path string, file *ast.File) {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if ok && fn.Recv != nil && forbiddenMethods[fn.Name.Name] {
				t.Errorf("%s: bootstrap must not own inbound operator method %s", path, fn.Name.Name)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := literal.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "cli" {
				switch selector.Sel.Name {
				case "AudienceDefinitionSummary", "AudienceSyncSummary", "InteractionRegion",
					"InteractionSurfaceSummary", "SurfaceDeploymentSummary", "PromotionSummary",
					"DeliveryIntentSummary", "AutomationRequestSummary":
					t.Errorf("%s: bootstrap constructs inbound DTO cli.%s", path, selector.Sel.Name)
				}
			}
			return true
		})
	})
}
