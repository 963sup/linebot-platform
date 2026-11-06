#!/usr/bin/env python3
"""Exercise the actual project stdio commands; this cannot certify host registration."""
import json
import os
from pathlib import Path
import selectors
import signal
import subprocess
import sys
import time
import tomllib

root = Path(__file__).resolve().parents[2]


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def check(name, server, root=root):
    require(name in ('repomix', 'gopls'), f'Unsupported probe: {name}')
    require(server.get('enabled'), f'{name} is disabled')
    require(server.get('command') and not server.get('url'), f'{name}: expected stdio command')
    env = os.environ | server.get('env', {})
    process = subprocess.Popen([server['command'], *server.get('args', [])], cwd=root,
                               env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               start_new_session=True)
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    buffer = b''
    sequence = 0

    def send(message):
        process.stdin.write((json.dumps({'jsonrpc': '2.0', **message}) + '\n').encode())
        process.stdin.flush()

    def request(method, params):
        nonlocal buffer, sequence
        sequence += 1
        send({'id': sequence, 'method': method, 'params': params})
        timeout = server.get('startup_timeout_sec', 30) if method == 'initialize' else server.get('tool_timeout_sec', 60)
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if b'\n' not in buffer:
                if not selector.select(max(0, deadline - time.monotonic())):
                    break
                data = os.read(process.stdout.fileno(), 65536)
                if not data:
                    raise RuntimeError(f'{name}: exited before {method} response')
                buffer += data
                continue
            line, buffer = buffer.split(b'\n', 1)
            message = json.loads(line)
            if message.get('method') == 'roots/list':
                send({'id': message['id'], 'result': {'roots': [{'uri': root.as_uri(), 'name': root.name}]}})
            elif message.get('id') == sequence:
                if 'error' in message:
                    raise RuntimeError(f'{name}: {message["error"]}')
                return message['result']
        raise TimeoutError(f'{name}: {method} timed out')

    try:
        request('initialize', {'protocolVersion': '2025-03-26', 'capabilities': {'roots': {}},
                              'clientInfo': {'name': 'project-smoke', 'version': '1'}})
        send({'method': 'notifications/initialized'})
        tools = request('tools/list', {})['tools']
        names = {tool['name'] for tool in tools}
        tool = 'file_system_read_file' if name == 'repomix' else 'go_workspace'
        require(tool in names, f'{name}: required tool {tool} not advertised')
        require(tool in server.get('enabled_tools', [tool]) and tool not in server.get('disabled_tools', []),
                f'{name}: {tool} blocked by tool policy')
        arguments = {'path': 'mise.toml'} if name == 'repomix' else {}
        result = request('tools/call', {'name': tool, 'arguments': arguments})
        require(not result.get('isError'), f'{name}: tool error {result}')
        require(result.get('content'), f'{name}: empty tool response')
        if name == 'repomix':
            require('min_version' in json.dumps(result), 'Repomix did not read expected file')
        print(f'{name}: initialize, tools/list ({len(tools)}), {tool} passed', flush=True)
    finally:
        selector.close()
        try:
            os.killpg(process.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            pass
        finally:
            # Kill any surviving children too, even if the direct process exited.
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
            process.stdin.close()
            process.stdout.close()


if __name__ == '__main__':
    selected = sys.argv[1:] or ['repomix', 'gopls']
    config = tomllib.loads((root / '.codex/config.toml').read_text())
    for name in selected:
        require(name in ('repomix', 'gopls'), f'Unsupported probe: {name}')
        check(name, config['mcp_servers'][name])
