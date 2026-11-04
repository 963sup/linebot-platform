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

	"github.com/963sup/linebot-platform/internal/application/usecases/channel"
	"github.com/963sup/linebot-platform/internal/bootstrap"
)

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
	var cmd channel.TokenCommand
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		return channel.TokenCommand{}, errors.New("invalid private token request JSON")
	}
	var additional any
	if err := dec.Decode(&additional); err != io.EOF {
		return channel.TokenCommand{}, errors.New("extra token request content")
	}
	return cmd, nil
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
	invoke := func() ([]byte, error) {
		rt, err := bootstrap.NewLINETokenOpsRuntime()
		if err != nil {
			return nil, err
		}
		defer rt.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		return rt.Tokens.Execute(ctx, cmd)
	}
	if isRevoke {
		_, err := invoke()
		return err
	}
	return writePrivateOperation(*out, invoke)
}
