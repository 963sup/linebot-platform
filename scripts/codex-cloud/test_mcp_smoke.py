"""Regression checks for the stdlib-only MCP probe, without external servers."""
import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock

import mcp_smoke

FAKE_SERVER = '''
import json, os, sys, time
mode = sys.argv[1]
if mode == 'timeout':
    open(sys.argv[2], 'w').write(str(os.getpid()))
    time.sleep(30)
for line in sys.stdin:
    msg = json.loads(line)
    if 'id' not in msg: continue
    method = msg['method']
    if mode == 'rpc-error':
        print(json.dumps({'jsonrpc':'2.0', 'id':msg['id'], 'error':{'code':-32603,'message':'failed'}}), flush=True)
        continue
    result = {}
    if method == 'tools/list': result = {'tools': [{'name': 'generate_skill' if mode == 'skill-no-files' else 'file_system_read_file'}]}
    if method == 'tools/list' and mode.startswith('graph-'):
        result = {'tools': [{'name': 'graph_stats'}, {'name': 'query_graph'}]}
    if method == 'tools/call' and mode.startswith('graph-'):
        if msg['params']['name'] == 'graph_stats':
            text = 'Nodes: 0' if mode == 'graph-empty' else 'Nodes: 42'
        else:
            text = 'No results' if mode == 'graph-no-match' else ('Ambiguous: validateConfig' if mode == 'graph-ambiguous' else 'NODE validateConfig() [src=cmd/codexcheck/main.go loc=L139]')
        result = {'content': [{'type': 'text', 'text': text}]}
        if mode == 'graph-error': result['isError'] = True
    elif method == 'tools/call':
        if mode != 'skill-no-files' and msg['params']['arguments'] != {'path': 'mise.toml'}: raise ValueError('unexpected read path')
        result = {'content':[{'type':'text','text':'min_version = "2026.11.3"'}]}
        if mode == 'tool-error': result['isError'] = True
        if mode == 'wrong-content': result['content'][0]['text'] = 'wrong file'
    print(json.dumps({'jsonrpc':'2.0','id':msg['id'],'result':result}), flush=True)
'''


class SmokeTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.fake = self.root / 'server.py'
        self.fake.write_text(FAKE_SERVER)
        self.pid = self.root / 'pid'

    def server(self, mode):
        return {'enabled': True, 'command': sys.executable,
                'args': [str(self.fake), mode, str(self.pid)],
                'startup_timeout_sec': 0.5, 'tool_timeout_sec': 0.5}

    def test_success(self):
        with contextlib.redirect_stdout(io.StringIO()) as output:
            mcp_smoke.check('repomix', self.server('success'), self.root)
        self.assertIn('file_system_read_file passed', output.getvalue())

    def test_generation_requires_real_artifacts_and_cleans_fixture(self):
        with self.assertRaisesRegex(RuntimeError, 'did not create SKILL.md'):
            mcp_smoke.check('repomix_skills', self.server('skill-no-files'), self.root)
        self.assertEqual(list(self.root.glob('.repomix-skill-smoke-*')), [])

    def test_protocol_and_tool_failures(self):
        for mode in ('rpc-error', 'tool-error', 'wrong-content'):
            with self.subTest(mode=mode), self.assertRaises(RuntimeError):
                mcp_smoke.check('repomix', self.server(mode), self.root)

    @mock.patch.object(mcp_smoke, "check_asyncio_ipc")
    def test_graphify_requires_nonempty_graph_and_symbol_query(self, _ipc):
        with contextlib.redirect_stdout(io.StringIO()):
            mcp_smoke.check('graphify', self.server('graph-success'), self.root)
        for mode in ('graph-empty', 'graph-no-match', 'graph-error', 'graph-ambiguous'):
            with self.subTest(mode=mode), self.assertRaises(RuntimeError):
                mcp_smoke.check('graphify', self.server(mode), self.root)

    def test_graphify_ipc_denial_fails_before_launch(self):
        sender, receiver = mock.MagicMock(), mock.MagicMock()
        sender.sendall.side_effect = PermissionError('Operation not permitted')
        with mock.patch.object(mcp_smoke.socket, 'socketpair', return_value=(sender, receiver)), \
             mock.patch.object(mcp_smoke.subprocess, 'Popen') as launch:
            with self.assertRaisesRegex(RuntimeError, 'local socket IPC unavailable'):
                mcp_smoke.check('graphify', self.server('graph-success'), self.root)
            launch.assert_not_called()
        sender.__exit__.assert_called_once()
        receiver.__exit__.assert_called_once()

    def test_timeout_reaps_child(self):
        start = time.monotonic()
        with self.assertRaises(TimeoutError):
            mcp_smoke.check('repomix', self.server('timeout'), self.root)
        self.assertLess(time.monotonic() - start, 3)
        with self.assertRaises(ProcessLookupError):
            os.kill(int(self.pid.read_text()), 0)

    def test_optimized_python_still_rejects_tool_errors(self):
        code = 'import mcp_smoke; mcp_smoke.check("repomix", ' + repr(self.server('tool-error')) + ')'
        result = subprocess.run([sys.executable, '-O', '-c', code],
                                cwd=Path(mcp_smoke.__file__).parent, capture_output=True, timeout=3)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(b'tool error', result.stderr)

    def test_disabled_or_filtered_tool_is_rejected(self):
        for changes in ({'enabled': False}, {'enabled_tools': []},
                        {'disabled_tools': ['file_system_read_file']}):
            with self.subTest(changes=changes), self.assertRaises(RuntimeError):
                mcp_smoke.check('repomix', self.server('success') | changes, self.root)


if __name__ == '__main__':
    unittest.main()
