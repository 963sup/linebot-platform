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

	"github.com/963sup/linebot-platform/internal/application/ports/outbound"
	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
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

func tokenRequestFromPrivateFile(path string) (channel.TokenCommand, error) {
	if path == "" {
		return channel.TokenCommand{}, errors.New("private token request file is required")
	}
	info, err := os.Lstat(filepath.Clean(path))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return channel.TokenCommand{}, errors.New("token request must be a regular file readable only by its owner")
	}
	data, err := readPrivateBounded(path, 256<<10)
	if err != nil {
		return channel.TokenCommand{}, err
	}
	var request tokenRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		return channel.TokenCommand{}, errors.New("invalid private token request JSON")
	}
	var additional any
	if err := dec.Decode(&additional); err != io.EOF {
		return channel.TokenCommand{}, errors.New("extra token request content")
	}
	return channel.TokenCommand{
		GrantType:           request.GrantType,
		ClientID:            request.ClientID,
		ClientSecret:        request.ClientSecret,
		ClientAssertionType: request.ClientAssertionType,
		ClientAssertion:     request.ClientAssertion,
		AccessToken:         request.AccessToken,
	}, nil
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
	if err := channel.ValidateTokenCommand(cmd); err != nil {
		return err
	}
	invoke := func() (outbound.LINEResult, error) {
		if tokenRuntimeFactory == nil {
			return outbound.LINEResult{}, errors.New("token runtime is not configured")
		}
		rt, err := tokenRuntimeFactory()
		if err != nil {
			return outbound.LINEResult{}, err
		}
		defer rt.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		return rt.Execute(ctx, cmd)
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
		payload, err := json.Marshal(result.Value)
		if err != nil || string(payload) == "null" {
			return nil, errors.New("invalid LINE token result")
		}
		return payload, nil
	})
}
