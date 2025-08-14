// File that queries notion to get the list of the safes + values that need to be monitored.
package multisig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const notionAPIVersion = "2022-06-28"

type NotionProps struct {
	// Column/property names in your Notion database
	Name           string // e.g. "Name"
	Address        string // e.g. "address (evm)"
	MultisigLead   string // e.g. "multisig lead"
	Risk           string // e.g. "risk"
	Networks       string // e.g. "Networks"
	SignerCount    string // e.g. "Signer count"
	Threshold      string // e.g. "threshold"
	Signers        string // e.g. "Signers"
	HasMonitoring  string // e.g. "Has Monitoring"
	HasBackupChat  string // e.g. "Has Backup Chat"
	LastReviewedby string // e.g. "Last Reviewed by"
	LastReviewDate string // e.g. "Last Review Date"
}

// Defaults to your provided schema (exact names) otherwise this will panic need make a
func DefaultNotionProps() NotionProps {
	return NotionProps{
		Name:           "Name",
		Address:        "Address",
		MultisigLead:   "Multisig Lead",
		Risk:           "Risk Band",
		Networks:       "Networks",
		SignerCount:    "Signer Count",
		Threshold:      "Threshold",
		Signers:        "Signers",
		HasMonitoring:  "Has Monitoring",
		HasBackupChat:  "Has Backup Chat",
		LastReviewedby: "Last Reviewed By",
		LastReviewDate: "Last Review Date",
	}
}

type NotionSafeRow struct {
	Name         string
	Address      string
	MultisigLead []string
	Risk         string
	Networks     []string
	SignerCount  int
	Threshold    int
}

// QueryNotionSafes pulls all rows from a Notion database and extracts the specific columns.
// token: Notion integration token (secret_...)
// databaseID: Notion database ID (32-char id or with hyphens)
// props: mapping of property names to read
func QueryNotionSafes(ctx context.Context, token, databaseID string, props NotionProps) ([]NotionSafeRow, error) {
	var out []NotionSafeRow
	client := http.DefaultClient
	// curl -s -X POST "https://api.notion.com/v1/databases/24ffb7d8d2cc80e8885ee1bb3bc1f53b/query" \
	//  -H "Authorization: Bearer $NOTION_TOKEN" \
	// -H "Notion-Version: 2022-06-28" \
	// -H "Content-Type: application/json" | jq

	endpoint := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", databaseID)

	type queryReq struct {
		PageSize    int     `json:"page_size,omitempty"`
		StartCursor *string `json:"start_cursor,omitempty"`
	}
	type queryResp struct {
		Results    []notionPage `json:"results"`
		HasMore    bool         `json:"has_more"`
		NextCursor *string      `json:"next_cursor"`
	}

	var cursor *string
	for {
		reqBody := queryReq{PageSize: 100, StartCursor: cursor} // Number of Multisig should be less than 100 anyway so this can be set here.
		payload, _ := json.Marshal(reqBody)
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Notion-Version", notionAPIVersion)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)

		if err != nil {
			return nil, err
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				data, _ := io.ReadAll(resp.Body)
				err = fmt.Errorf("notion http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
				return
			}
			var parsed queryResp
			if e := json.NewDecoder(resp.Body).Decode(&parsed); e != nil {
				err = e
				return
			}

			for _, page := range parsed.Results {
				rec, ok := extractRow(page, props)
				if ok {
					out = append(out, rec)
				}
			}
			if !parsed.HasMore || parsed.NextCursor == nil || *parsed.NextCursor == "" {
				cursor = nil
			} else {
				cursor = parsed.NextCursor
			}
		}()
		if err != nil {
			return nil, err
		}
		if cursor == nil {
			break
		}
	}
	return out, nil
}

type notionPage struct {
	Properties map[string]json.RawMessage `json:"properties"`
}

type notionProperty struct {
	Type        string           `json:"type"`
	Title       []notionRichText `json:"title,omitempty"`
	RichText    []notionRichText `json:"rich_text,omitempty"`
	Number      *float64         `json:"number,omitempty"`
	URL         *string          `json:"url,omitempty"`
	Select      *notionSelect    `json:"select,omitempty"`
	MultiSelect []notionSelect   `json:"multi_select,omitempty"`
	People      []notionUser     `json:"people,omitempty"`
}

type notionRichText struct {
	PlainText string `json:"plain_text"`
}

type notionSelect struct {
	Name string `json:"name"`
}

type notionUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func extractRow(pg notionPage, props NotionProps) (NotionSafeRow, bool) {
	getProp := func(key string) (notionProperty, bool) {
		raw, ok := pg.Properties[key]
		if !ok {
			return notionProperty{}, false
		}
		var p notionProperty
		if err := json.Unmarshal(raw, &p); err != nil {
			return notionProperty{}, false
		}
		return p, true
	}
	// need to check if the property is nil or not for each property in the future.
	pName, _ := getProp(props.Name)
	pAddr, _ := getProp(props.Address)
	pLead, _ := getProp(props.MultisigLead)
	pRisk, _ := getProp(props.Risk)
	pNets, _ := getProp(props.Networks)
	pSigners, errSignerCounts := getProp(props.SignerCount)
	if errSignerCounts {
		fmt.Println("pSigners", *pSigners.Number)
	}
	pThr, _ := getProp(props.Threshold)
	//pHasMonitoring, _ := getProp(props.HasMonitoring)
	//pHasBackupChat, _ := getProp(props.HasBackupChat)
	//pLastReviewedby, _ := getProp(props.LastReviewedby)
	//pLastReviewDate, _ := getProp(props.LastReviewDate)
	//fmt.Println("pName", pName)
	//fmt.Println("pAddr", pAddr)
	//fmt.Println("pLead", pLead)
	//fmt.Println("pRisk", pRisk)
	//fmt.Println("pNets", pNets)
	//fmt.Println("pSigners", pSigners)
	//fmt.Println("pThr", pThr)
	// Permit partial rows: require at least Address + Threshold
	// if !(ok2 && ok7) {
	// 	return NotionSafeRow{}, false
	// }

	name := extractText(pName)
	addr := strings.TrimSpace(extractTextPrefer(pAddr, []string{"title", "rich_text", "url", "select"}))
	lead := extractPeopleOrTextList(pLead)
	risk := extractTextPrefer(pRisk, []string{"select", "title", "rich_text"})
	nets := extractMultiSelectOrList(pNets)
	signers := extractInt(pSigners)
	thr := extractInt(pThr)

	// basic validation
	if addr == "" || thr < 0 {
		return NotionSafeRow{}, false
	}

	return NotionSafeRow{
		Name:         name,
		Address:      addr,
		MultisigLead: lead,
		Risk:         risk,
		Networks:     nets,
		SignerCount:  signers,
		Threshold:    thr,
	}, true
}

func extractText(p notionProperty) string {
	if strings.EqualFold(p.Type, "title") && len(p.Title) > 0 {
		return strings.TrimSpace(p.Title[0].PlainText)
	}
	if strings.EqualFold(p.Type, "rich_text") && len(p.RichText) > 0 {
		return strings.TrimSpace(p.RichText[0].PlainText)
	}
	if strings.EqualFold(p.Type, "url") && p.URL != nil {
		return strings.TrimSpace(*p.URL)
	}
	if strings.EqualFold(p.Type, "select") && p.Select != nil {
		return strings.TrimSpace(p.Select.Name)
	}
	return ""
}

func extractTextPrefer(p notionProperty, order []string) string {
	for _, t := range order {
		switch {
		case strings.EqualFold(p.Type, "title") && t == "title" && len(p.Title) > 0:
			return strings.TrimSpace(p.Title[0].PlainText)
		case strings.EqualFold(p.Type, "rich_text") && t == "rich_text" && len(p.RichText) > 0:
			return strings.TrimSpace(p.RichText[0].PlainText)
		case strings.EqualFold(p.Type, "url") && t == "url" && p.URL != nil:
			return strings.TrimSpace(*p.URL)
		case strings.EqualFold(p.Type, "select") && t == "select" && p.Select != nil:
			return strings.TrimSpace(p.Select.Name)
		}
	}
	return extractText(p)
}

func extractMultiSelectOrList(p notionProperty) []string {
	if strings.EqualFold(p.Type, "multi_select") && len(p.MultiSelect) > 0 {
		out := make([]string, 0, len(p.MultiSelect))
		for _, it := range p.MultiSelect {
			if s := strings.TrimSpace(it.Name); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	// fallback to comma-separated rich_text/title
	if s := extractText(p); s != "" {
		chunks := strings.Split(s, ",")
		out := make([]string, 0, len(chunks))
		for _, c := range chunks {
			if v := strings.TrimSpace(c); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}

func extractPeopleOrTextList(p notionProperty) []string {
	if strings.EqualFold(p.Type, "people") && len(p.People) > 0 {
		out := make([]string, 0, len(p.People))
		for _, u := range p.People {
			if name := strings.TrimSpace(u.Name); name != "" {
				out = append(out, name)
			} else if id := strings.TrimSpace(u.ID); id != "" {
				out = append(out, id)
			}
		}
		return out
	}
	return extractMultiSelectOrList(p)
}

func extractInt(p notionProperty) int {
	if strings.EqualFold(p.Type, "number") && p.Number != nil {
		return int(*p.Number)
	}
	// fallback to parsing text
	if s := extractText(p); s != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return v
		}
	}
	return -1
}

// IsWebhookEnabled checks if a webhook URL is valid and properly configured
func IsWebhookEnabled(webhookURL string) bool {
	if webhookURL == "" {
		return false
	}

	// Validate URL format
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return false
	}

	// Check if URL has proper scheme and host
	if parsedURL.Scheme == "" {
		return false
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}
	if parsedURL.Host == "" {
		return false
	}

	return true
}

// SendWebhookAlert sends a simple string message to a webhook URL
func SendWebhookAlert(webhookURL, message string) error {
	// Create simple JSON payload
	payload := map[string]interface{}{
		"content":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send HTTP POST request
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check if request was successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
