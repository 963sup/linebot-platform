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
    if method == 'tools/list': result = {'tools': [{'name':'file_system_read_file'}]}
    if method == 'tools/call':
        if msg['params']['arguments'] != {'path': 'mise.toml'}: raise ValueError('expected sandbox-relative path')
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

    def test_protocol_and_tool_failures(self):
        for mode in ('rpc-error', 'tool-error', 'wrong-content'):
            with self.subTest(mode=mode), self.assertRaises(RuntimeError):
                mcp_smoke.check('repomix', self.server(mode), self.root)

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
