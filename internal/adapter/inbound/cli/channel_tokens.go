package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	credentiallifecycle "github.com/963sup/linebot-platform/internal/application/usecases/credential-lifecycle"
)

// tokenRequest is the private-file wire format. Secret-bearing JSON metadata
// stays at the CLI boundary and never enters the application package.
type tokenRequest struct {
	GrantType           string `json:"grantType,omitempty"`
	ClientID            string `json:"clientId,omitempty"`
	ClientSecret        string `json:"clientSecret,omitempty"`
	ClientAssertionType string `json:"clientAssertionType,omitempty"`
	ClientAssertion     string `json:"clientAssertion,omitempty"`
	AccessToken         string `json:"accessToken,omitempty"`
}

type tokenCommand struct {
	Action              string
	GrantType           string
	ClientID            string
	ClientSecret        string
	ClientAssertionType string
	ClientAssertion     string
	AccessToken         string
}

func tokenRequestFromPrivateFile(path string) (tokenCommand, error) {
	if path == "" {
		return tokenCommand{}, errors.New("private token request file is required")
	}
	info, err := os.Lstat(filepath.Clean(path))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return tokenCommand{}, errors.New("token request must be a regular file readable only by its owner")
	}
	data, err := readPrivateBounded(path, 256<<10)
	if err != nil {
		return tokenCommand{}, err
	}
	var request tokenRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		return tokenCommand{}, errors.New("invalid private token request JSON")
	}
	var additional any
	if err := dec.Decode(&additional); err != io.EOF {
		return tokenCommand{}, errors.New("extra token request content")
	}
	return tokenCommand{
		GrantType:           request.GrantType,
		ClientID:            request.ClientID,
		ClientSecret:        request.ClientSecret,
		ClientAssertionType: request.ClientAssertionType,
		ClientAssertion:     request.ClientAssertion,
		AccessToken:         request.AccessToken,
	}, nil
}

func validateTokenCommand(command tokenCommand) error {
	switch command.Action {
	case "issue-stateless":
		if command.GrantType == "" || ((command.ClientID == "" || command.ClientSecret == "") && (command.ClientAssertionType == "" || command.ClientAssertion == "")) {
			return errors.New("issue-stateless requires a grant and one credential form")
		}
	case "key-ids", "issue-jwt":
		if command.ClientAssertionType == "" || command.ClientAssertion == "" || (command.Action == "issue-jwt" && command.GrantType == "") {
			return errors.New("JWT operation requires assertion credentials")
		}
	case "verify-jwt", "verify-short", "revoke-short":
		if command.AccessToken == "" {
			return errors.New("access token is required")
		}
	case "revoke-jwt":
		if command.ClientID == "" || command.ClientSecret == "" || command.AccessToken == "" {
			return errors.New("revoke-jwt requires client credentials and token")
		}
	case "issue-short":
		if command.GrantType == "" || command.ClientID == "" || command.ClientSecret == "" {
			return errors.New("issue-short requires client credentials")
		}
	default:
		return errors.New("unsupported token action")
	}
	return nil
}

// runChannelTokens is available ONLY in the local ops CLI. It never accepts
// secrets on command line flags, never prints them, and never emits credentials
// to structured logs. All results containing a token go to an exclusive file.
func runChannelTokens(args []string) error {
	flags := flag.NewFlagSet("channel-tokens", flag.ContinueOnError)
	action := flags.String("action", "", "issue-stateless|key-ids|issue-jwt|verify-jwt|revoke-jwt|issue-short|verify-short|revoke-short")
	request := flags.String("request", "", "owner-readable (0600) JSON file holding credentials")
	out := flags.String("out", "", "exclusive private output file (0600)")
	read := flags.Bool("read", false, "authorize query")
	apply := flags.Bool("apply", false, "authorize issuance or revocation")
	confirm := flags.String("confirm", "", "must equal action for mutation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || *request == "" {
		return errors.New("channel-tokens requires --request and no extra arguments")
	}
	isRead := *action == "key-ids" || *action == "verify-jwt" || *action == "verify-short"
	isRevoke := *action == "revoke-jwt" || *action == "revoke-short"
	if isRead {
		if !*read || *apply || *confirm != "" || *out == "" {
			return errors.New("token read requires --read and --out")
		}
	} else if !*apply || *read || *confirm == "" || *confirm != *action {
		return errors.New("token mutation requires --apply --confirm ACTION")
	}
	if isRevoke && *out != "" {
		return errors.New("revoke does not produce an output file")
	}
	if !isRead && !isRevoke && *out == "" {
		return errors.New("issued token requires private --out")
	}
	cmd, err := tokenRequestFromPrivateFile(*request)
	if err != nil {
		return err
	}
	cmd.Action = *action
	if err := validateTokenCommand(cmd); err != nil {
		return err
	}
	invoke := func() (any, error) {
		if tokenRuntimeFactory == nil {
			return nil, errors.New("token runtime is not configured")
		}
		rt, err := tokenRuntimeFactory()
		if err != nil {
			return nil, err
		}
		defer rt.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		client := credentiallifecycle.ClientCredential{ClientID: cmd.ClientID, ClientSecret: cmd.ClientSecret}
		assertion := credentiallifecycle.AssertionCredential{Type: cmd.ClientAssertionType, Assertion: cmd.ClientAssertion}
		switch cmd.Action {
		case "issue-stateless":
			if cmd.ClientID != "" {
				return rt.IssueStatelessWithClient(ctx, cmd.GrantType, client)
			}
			return rt.IssueStatelessWithAssertion(ctx, cmd.GrantType, assertion)
		case "key-ids":
			return rt.ValidKeyIDs(ctx, assertion)
		case "issue-jwt":
			return rt.IssueJWT(ctx, cmd.GrantType, assertion)
		case "verify-jwt":
			return rt.VerifyJWT(ctx, cmd.AccessToken)
		case "revoke-jwt":
			return nil, rt.RevokeJWT(ctx, client, cmd.AccessToken)
		case "issue-short":
			return rt.IssueShortLived(ctx, cmd.GrantType, client)
		case "verify-short":
			return rt.VerifyShortLived(ctx, cmd.AccessToken)
		case "revoke-short":
			return nil, rt.RevokeShortLived(ctx, cmd.AccessToken)
		default:
			return nil, errors.New("unsupported token action")
		}
	}
	if isRevoke {
		_, err := invoke()
		return err
	}
	return writePrivateOperation(*out, func() ([]byte, error) {
		result, err := invoke()
		if err != nil {
			return nil, err
		}
		payload, err := marshalCredentialResult(result)
		if err != nil || string(payload) == "null" {
			return nil, errors.New("invalid LINE token result")
		}
		return payload, nil
	})
}

func marshalCredentialResult(result any) ([]byte, error) {
	switch value := result.(type) {
	case credentiallifecycle.IssuedCredential:
		return json.Marshal(struct {
			AccessToken string    `json:"access_token"`
			TokenType   string    `json:"token_type,omitempty"`
			ExpiresAt   time.Time `json:"expires_at,omitempty"`
		}{value.AccessToken, value.TokenType, value.ExpiresAt})
	case credentiallifecycle.Verification:
		return json.Marshal(struct {
			Active    bool      `json:"active"`
			ClientID  string    `json:"client_id,omitempty"`
			ExpiresAt time.Time `json:"expires_at,omitempty"`
		}{value.Active, value.ClientID, value.ExpiresAt})
	case []string:
		return json.Marshal(struct {
			Kids []string `json:"kids"`
		}{value})
	default:
		return nil, errors.New("invalid LINE token result")
	}
}
