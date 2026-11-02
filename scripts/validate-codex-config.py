#!/usr/bin/env python3
"""Dependency-free validation of checked-in Codex config, agents and cloud setup."""
from pathlib import Path
import subprocess
import sys
import tomllib

ROOT = Path(__file__).resolve().parent.parent
PROJECT = ROOT / ".codex"
EXPECTED_AGENTS = {
    "domain_architect",
    "line_integrator",
    "cloud_platform",
    "go_reviewer",
    "docs_researcher",
}
EXPECTED_SERVERS = {"gopls", "context7", "github_readonly", "firestore_toolbox"}


def check() -> None:
    with (PROJECT / "config.toml").open("rb") as stream:
        config = tomllib.load(stream)

    assert config["agents"]["enabled"] is True
    assert config["agents"]["max_concurrent_threads_per_session"] <= 3
    assert config["features"]["multi_agent"] is True

    servers = config["mcp_servers"]
    assert set(servers) == EXPECTED_SERVERS
    assert servers["gopls"]["enabled"] is True
    assert servers["gopls"]["command"] == "bash"
    assert servers["context7"]["url"] == "https://mcp.context7.com/mcp"
    assert servers["github_readonly"]["enabled"] is False
    assert servers["github_readonly"]["bearer_token_env_var"] == "GITHUB_MCP_TOKEN"
    assert servers["github_readonly"]["http_headers"]["X-MCP-Readonly"] == "true"
    assert servers["firestore_toolbox"]["enabled"] is False
    assert servers["firestore_toolbox"]["args"][-2:] == ["--prebuilt=firestore", "--stdio"]
    assert set(servers["firestore_toolbox"]["enabled_tools"]) == {
        "get_documents", "query_collection", "list_collections",
        "get_rules", "validate_rules",
    }
    assert all(server["required"] is False for server in servers.values())

    actual = {}
    for file in sorted((PROJECT / "agents").glob("*.toml")):
        with file.open("rb") as stream:
            role = tomllib.load(stream)
        for field in ("name", "description", "developer_instructions"):
            assert isinstance(role.get(field), str) and role[field].strip(), (
                file, field
            )
        assert role["name"] == file.stem
        # Role files inherit the parent session's security, provider and MCP.
        assert not {"mcp_servers", "approval_policy", "sandbox_mode"} & role.keys()
        assert file.stem not in actual
        actual[file.stem] = role
    assert set(actual) == EXPECTED_AGENTS, set(actual)

    setup = ROOT / "scripts/setup-codex-cloud.sh"
    assert setup.exists()
    subprocess.run(["bash", "-n", str(setup)], check=True)

    print(f"Codex config OK: {len(actual)} agents, {len(servers)} MCP entries")


if __name__ == "__main__":
    try:
        check()
    except (AssertionError, OSError, subprocess.CalledProcessError, ValueError, KeyError) as exc:
        print(f"Codex config verification failed: {exc}", file=sys.stderr)
        sys.exit(1)
