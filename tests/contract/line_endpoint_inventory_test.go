package contract

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

// The pinned LINE OpenAPI snapshot must remain an explicit, unique inventory.
// This detects accidental omissions and dishonest integration status changes;
// it is not a live LINE endpoint or channel-permission test.
func TestLINEEndpointInventory(t *testing.T) {
	file, err := os.Open("../../docs/line/endpoint-inventory.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	item := regexp.MustCompile(`^  - \{source: ([a-z-]+), group: ([a-z-]+), operation: ([A-Za-z0-9]+), method: (GET|POST|PUT|DELETE|PATCH), path: "([^"]+)", integration: (not_integrated|integrated_offline), test: (none|contract), access: (review|required_eligibility_review)\}$`)
	implemented := map[string]bool{
		"replyMessage":                     false,
		"pushMessage":                      false,
		"getProfile":                       false,
		"getBotInfo":                       false,
		"getWebhookEndpoint":               false,
		"getMessageQuota":                  false,
		"getMessageQuotaConsumption":       false,
		"getAggregationUnitUsage":          false,
		"getAggregationUnitNameList":       false,
		"getNumberOfSentReplyMessages":     false,
		"getNumberOfSentPushMessages":      false,
		"getNumberOfSentMulticastMessages": false,
		"getNumberOfSentBroadcastMessages": false,
		"getNarrowcastProgress":            false,
		"getRichMenuList":                  false,
		"getRichMenu":                      false,
		"getDefaultRichMenuId":             false,
		"getRichMenuAliasList":             false,
		"getRichMenuAlias":                 false,
		"getRichMenuIdOfUser":              false,
		"getRichMenuBatchProgress":         false,
		"getFollowers":                     false,
		"getGroupSummary":                  false,
		"getGroupMemberCount":              false,
		"getRoomMemberCount":               false,
		"getGroupMembersIds":               false,
		"getRoomMembersIds":                false,
		"getGroupMemberProfile":            false,
		"getRoomMemberProfile":             false,
		"getFriendsDemographics":           false,
		"getNumberOfMessageDeliveries":     false,
		"getNumberOfFollowers":             false,
		"getMessageEvent":                  false,
		"getStatisticsPerUnit":             false,
		"getRichMenuInsightSummary":        false,
		"getRichMenuInsightDaily":          false,
		"getMembershipList":                false,
		"getMembershipSubscription":        false,
		"getJoinedMembershipUsers":         false,
		"listCoupon":                       false,
		"getCouponDetail":                  false,
		"getAudienceData":                  false,
		"getSharedAudienceData":            false,
		"getAudienceGroups":                false,
		"getSharedAudienceGroups":          false,
		"setDefaultRichMenu":               false,
		"cancelDefaultRichMenu":            false,
		"linkRichMenuIdToUser":             false,
		"unlinkRichMenuIdFromUser":         false,
		"deleteRichMenu":                   false,
		"deleteRichMenuAlias":              false,
		"broadcast": false,
		"multicast": false,
	}
	seen := make(map[string]bool)
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "  - ") {
			continue
		}
		groups := item.FindStringSubmatch(line)
		if groups == nil {
			t.Fatalf("malformed endpoint entry: %s", line)
		}
		key := groups[4] + " " + groups[5]
		if seen[key] {
			t.Fatalf("duplicate endpoint: %s", key)
		}
		seen[key] = true
		count++
		if groups[6] == "integrated_offline" {
			if _, ok := implemented[groups[3]]; !ok || groups[7] != "contract" {
				t.Fatalf("unverified integrated endpoint: %s", groups[3])
			}
			implemented[groups[3]] = true
		} else if groups[7] != "none" {
			t.Fatalf("unintegrated endpoint claims tests: %s", groups[3])
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 102 {
		t.Fatalf("inventory count = %d, want pinned upstream baseline 102", count)
	}
	for op, ok := range implemented {
		if !ok {
			t.Fatalf("missing implemented endpoint: %s", op)
		}
	}
}
