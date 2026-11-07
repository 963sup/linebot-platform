package cli

import (
	"context"
	"errors"
	"flag"
	"time"
)

func runMenuImage(parent context.Context, args []string, getenv func(string) string, dependencies ...Dependencies) (err error) {
	deps := dependency(dependencies)
	flags := flag.NewFlagSet("menu-image", flag.ContinueOnError)
	id := flags.String("menu-id", "", "rich menu ID")
	out := flags.String("out", "", "new private output file")
	read := flags.Bool("read", false, "authorize retrieving the menu image")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) != 0 || !*read || *id == "" || *out == "" || getenv == nil {
		return errors.New("menu-image requires --menu-id --out --read")
	}
	if deps.MenuImageRuntime == nil {
		return errors.New("menu image runtime is not configured")
	}
	r, err := deps.MenuImageRuntime(getenv(messagingAPIChannelAccessTokenEnv))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.Close()) }()
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	data, err := r.PresentationImage(ctx, *id)
	if err != nil {
		return err
	}
	return writePrivateFile(*out, data.Data)
}
