"""Host integration tests for disposable developer check workspaces."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
import uuid

SCRIPT = Path(__file__).with_name('check-workspace.py').resolve()
spec = importlib.util.spec_from_file_location('workspace', SCRIPT)
workspace = importlib.util.module_from_spec(spec)
spec.loader.exec_module(workspace)


class WorkspaceChecks(unittest.TestCase):
    def setUp(self):
        self.assertEqual(os.geteuid(), 0, 'run on Host with sudo; no skipped safety checks')
        raw = SCRIPT.parent.parent / 'docs/github-logs/system-recovery-20261105'
        self.fixture = Path(tempfile.mkdtemp(prefix='workspace-regression-', dir=raw))
        os.chown(self.fixture, 1000, 1000)
        self.repo = self.fixture / 'repo'
        self.repo.mkdir()
        os.chown(self.repo, 1000, 1000)
        (self.repo / 'input.txt').write_text('fixed source\n')
        os.chown(self.repo / 'input.txt', 1000, 1000)
        for args in [('init', '-q'), ('add', 'input.txt')]:
            subprocess.run(['runuser', '-u', 'winamp', '--', 'git', '-C', str(self.repo), *args], check=True)
        self.state = self.fixture / 'state'
        self.state.mkdir(mode=0o700)
        self.workspaces = []
        self.children = []

    def tearDown(self):
        for pid in self.children:
            try:
                os.killpg(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        for path in self.workspaces:
            if path.is_symlink():
                path.unlink()
            elif path.exists():
                shutil.rmtree(path)
        shutil.rmtree(self.fixture)

    def command(self, action, code=None):
        args = [sys.executable, str(SCRIPT), action, '--state', str(self.state), '--repo', str(self.repo)]
        if code is not None:
            args += ['--', sys.executable, '-c', code]
        return args

    def call(self, action, code=None):
        return subprocess.run(self.command(action, code), capture_output=True, text=True, timeout=30)

    def record(self):
        receipt = next(self.state.glob('*/result.json'))
        record = json.loads(receipt.read_text())
        self.workspaces.append(Path(record['workspace']))
        return receipt, record

    def seeded(self):
        task = self.state / uuid.uuid4().hex
        task.mkdir()
        root = Path('/tmp') / ('haa-work-check-' + task.name)
        root.mkdir(mode=0o700)
        os.chown(root, 1000, 1000)
        (root / 'payload').write_text('owned bytes')
        os.chown(root / 'payload', 1000, 1000)
        s = root.stat()
        record = {'workspace': str(root), 'uid': 1000, 'pid': 2147483647, 'start': 'gone',
                  'boot': Path('/proc/sys/kernel/random/boot_id').read_text().strip(),
                  'identity': [s.st_dev, s.st_ino, s.st_uid], 'cleanup': 'PENDING', 'command_exit': None}
        receipt = task / 'result.json'
        workspace.save(receipt, record)
        self.workspaces.append(root)
        return root, receipt, record

    def test_success_and_primary_failure_keep_logs_and_remove_payload(self):
        for exit_code in (0, 7):
            with self.subTest(exit_code=exit_code):
                result = self.call('run', f"import os,sys; print(os.getuid()); open('output','w').write('x'); sys.exit({exit_code})")
                self.assertEqual(result.returncode, exit_code, result.stderr + result.stdout)
        for receipt in self.state.glob('*/result.json'):
            record = json.loads(receipt.read_text())
            self.assertEqual(record['cleanup'], 'CONFIRMED_REMOVED')
            self.assertFalse(Path(record['workspace']).exists())
            self.assertEqual((receipt.parent / 'command.log').read_text().strip(), '1000')
        self.assertEqual(self.call('status').returncode, 0)

    def test_open_file_blocks_new_work_and_recovery_until_released(self):
        root, receipt, record = self.seeded()
        with (root / 'payload').open() as held:
            self.assertFalse(workspace.collect(record, receipt))
            self.assertTrue(record['references'])
            self.assertNotEqual(self.call('run', "print('must not run')").returncode, 0)
            self.assertNotEqual(self.call('recover').returncode, 0)
            self.assertTrue(root.exists())
        self.assertEqual(self.call('recover').returncode, 0)
        self.assertFalse(root.exists())

    def test_cleanup_failure_is_nonzero_and_keeps_primary_error(self):
        for primary in (0, 7):
            result = self.call('run', f"import os,sys; os.symlink('/etc/hostname','foreign'); sys.exit({primary})")
            self.assertEqual(result.returncode, primary or 1, result.stdout + result.stderr)
            receipt = Path(json.loads(result.stdout)['receipt'])
            record = json.loads(receipt.read_text())
            root = Path(record['workspace'])
            self.workspaces.append(root)
            self.assertEqual(record['command_exit'], primary)
            self.assertEqual(record['cleanup'], 'INCOMPLETE')
            self.assertEqual(record['source_files'], 1)
            self.assertTrue(record['source_digest'])
            self.assertNotEqual(self.call('status').returncode, 0)
            (root / 'source/foreign').unlink()
            self.assertEqual(self.call('recover').returncode, 0)

    def test_symlink_hardlink_and_identity_exchange_are_preserved(self):
        for kind in ('symlink', 'hardlink', 'identity'):
            with self.subTest(kind=kind):
                root, receipt, record = self.seeded()
                external = self.fixture / ('external-' + kind)
                external.write_text('protected')
                if kind == 'symlink':
                    (root / 'foreign').symlink_to(external)
                elif kind == 'hardlink':
                    os.link(root / 'payload', root / 'foreign')
                else:
                    record['identity'][1] += 1
                self.assertFalse(workspace.collect(record, receipt))
                self.assertTrue(root.exists())
                self.assertEqual(external.read_text(), 'protected')

    def test_query_error_and_incomplete_registration_fail_closed(self):
        root, receipt, record = self.seeded()
        with patch.object(workspace, 'references', side_effect=RuntimeError('query unavailable')):
            self.assertFalse(workspace.collect(record, receipt))
        self.assertTrue(root.exists())
        broken = self.state / uuid.uuid4().hex
        broken.mkdir()
        self.assertNotEqual(self.call('status').returncode, 0)
        self.assertNotEqual(self.call('run', "print('must not run')").returncode, 0)

    def start_sleeping(self):
        code = "import pathlib,time; pathlib.Path('ready').write_text('ready'); time.sleep(60)"
        child = subprocess.Popen(self.command('run', code), stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            receipts = list(self.state.glob('*/result.json'))
            if receipts:
                record = json.loads(receipts[0].read_text())
                root = Path(record['workspace'])
                if (root / 'source/ready').exists():
                    self.children.append(record['child_pid'])
                    self.workspaces.append(root)
                    return child, root
            if child.poll() is not None:
                self.fail(child.stderr.read())
            time.sleep(0.05)
        child.kill()
        child.wait()
        self.fail('child readiness timed out')

    def test_term_preserves_interruption_and_cleans(self):
        child, root = self.start_sleeping()
        child.terminate()
        self.assertEqual(child.wait(timeout=20), 143)
        self.assertFalse(root.exists())
        child.stderr.close()

    def test_killed_owner_requires_recovery_and_protects_live_child(self):
        child, root = self.start_sleeping()
        child.kill()
        child.wait(timeout=10)
        child.stderr.close()
        self.assertNotEqual(self.call('recover').returncode, 0)
        self.assertTrue(root.exists())
        os.killpg(self.children[-1], signal.SIGKILL)
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            result = self.call('recover')
            if result.returncode == 0:
                break
            time.sleep(0.05)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.assertFalse(root.exists())


if __name__ == '__main__':
    unittest.main()
