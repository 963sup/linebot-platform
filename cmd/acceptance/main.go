// Command acceptance is a fail-closed, read-only final staging evidence gate.
// It never deploys, calls paid APIs, reads credentials or marks a PR merged.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	file := flag.String("evidence", "", "path to non-committed reviewer evidence JSON")
	flag.Parse()
	if *file == "" {
		fmt.Fprintln(os.Stderr, "stage-5 acceptance: -evidence is required")
		os.Exit(2)
	}
	commit, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stage-5 acceptance: unable to verify repository HEAD")
		os.Exit(2)
	}
	e, err := loadEvidence(*file)
	if err == nil {
		err = checkStagingEvidence(e, strings.TrimSpace(string(commit)), time.Now().UTC())
	}
	if err != nil {
		// Never output the evidence JSON, URLs, credentials, service IDs or
		// provider error details in CI/agent logs.
		fmt.Fprintln(os.Stderr, "stage-5 acceptance is BLOCKED:", err)
		os.Exit(1)
	}
	fmt.Println("stage-5 reviewer evidence gate: PASS (reviewer-attested; not independently verified by this command)")
}
