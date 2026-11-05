#!/usr/bin/env python3
"""Run local checks in an owned disposable source/cache workspace (Linux Host).

The durable state directory is explicit. Arbitrary Host paths and Docker resources
are never collected. Commands must keep transient writes inside the workspace.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import pwd
import shutil
import signal
import stat
import subprocess
import sys
import time
import uuid


def save(path, value):
    temporary = path.with_suffix('.new')
    with temporary.open('w') as stream:
        json.dump(value, stream, indent=2)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, path)
    fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def process_identity(pid):
    try:
        body = Path(f'/proc/{pid}/stat').read_text()
        return body.rsplit(')', 1)[1].split()[19]
    except FileNotFoundError:
        return None


def references(root):
    """A query failure is a refusal, including inaccessible process namespaces."""
    hits = []
    for proc in Path('/proc').iterdir():
        if not proc.name.isdecimal():
            continue
        try:
            paths = []
            for entry in [proc / 'cwd', proc / 'exe', *(proc / 'fd').iterdir()]:
                try:
                    paths.append(os.readlink(entry).removesuffix(' (deleted)'))
                except FileNotFoundError:
                    pass
            for line in (proc / 'maps').read_text().splitlines():
                fields = line.split(None, 5)
                if len(fields) == 6:
                    paths.append(fields[5].removesuffix(' (deleted)'))
            for path in paths:
                if path == str(root) or path.startswith(str(root) + '/'):
                    hits.append({'pid': int(proc.name), 'path': path})
            for line in (proc / 'mountinfo').read_text().splitlines():
                fields = line.split()
                # Reject both mounts into this tree and bind mounts from it.
                for index in (3, 4):
                    path = fields[index].replace('\\040', ' ').replace('\\134', '\\')
                    if path == str(root) or path.startswith(str(root) + '/'):
                        hits.append({'pid': int(proc.name), 'mount': path})
        except (FileNotFoundError, ProcessLookupError):
            pass
        except OSError as exc:
            raise RuntimeError(f'process query incomplete: {proc.name}: {exc}') from exc
    return hits


def inventory(root, uid):
    logical = allocated = count = 0
    for base, directories, files in os.walk(root, followlinks=False):
        for name in ['.', *directories, *files]:
            path = Path(base) / name
            s = path.lstat()
            if s.st_uid != uid or not (stat.S_ISREG(s.st_mode) or stat.S_ISDIR(s.st_mode)):
                raise RuntimeError(f'foreign or special workspace member: {path}')
            if stat.S_ISREG(s.st_mode):
                if s.st_nlink != 1:
                    raise RuntimeError(f'linked workspace member: {path}')
                logical += s.st_size
                allocated += s.st_blocks * 512
                count += 1
    return {'logical_bytes': logical, 'allocated_file_bytes': allocated, 'files': count}


def collect(record, receipt):
    root = Path(record['workspace'])
    try:
        if root.parent != Path('/tmp') or root.name != 'haa-work-check-' + receipt.parent.name:
            raise RuntimeError('workspace path differs from registered task')
        if record['boot'] == Path('/proc/sys/kernel/random/boot_id').read_text().strip():
            if record['pid'] != os.getpid() and process_identity(record['pid']) == record['start']:
                raise RuntimeError('task owner is still active')
        if not root.exists() and not root.is_symlink():
            record['cleanup'] = 'CONFIRMED_ABSENT'
        else:
            s = root.lstat()
            identity = record.get('identity')
            if identity is None or [s.st_dev, s.st_ino, s.st_uid] != identity or not stat.S_ISDIR(s.st_mode):
                raise RuntimeError('workspace identity differs or creation was interrupted; preserve for review')
            hits = references(root)
            if hits:
                record['references'] = hits
                raise RuntimeError('workspace has active references or mounts')
            record['before_delete'] = inventory(root, record['uid'])
            record['cleanup'] = 'DELETING'
            save(receipt, record)
            # fd-based rmtree avoids following an exchanged directory symlink.
            if not shutil.rmtree.avoids_symlink_attacks:
                raise RuntimeError('safe directory removal is unavailable')
            if [root.stat().st_dev, root.stat().st_ino, root.stat().st_uid] != identity:
                raise RuntimeError('workspace changed before deletion')
            shutil.rmtree(root)
            if root.exists() or root.is_symlink():
                raise RuntimeError('workspace still exists after deletion')
            record['cleanup'] = 'CONFIRMED_REMOVED'
        record.pop('cleanup_error', None)
    except (OSError, RuntimeError) as exc:
        record['cleanup'] = 'INCOMPLETE'
        record['cleanup_error'] = str(exc)
    save(receipt, record)
    return record['cleanup'] in ('CONFIRMED_REMOVED', 'CONFIRMED_ABSENT')


def pending(state, recover=False):
    unresolved = []
    for task in sorted(state.iterdir()):
        if task.name == 'lock':
            continue
        receipt = task / 'result.json'
        if task.is_symlink() or not task.is_dir() or not receipt.is_file() or receipt.is_symlink():
            unresolved.append(str(task))
            continue
        record = json.loads(receipt.read_text())
        if record['cleanup'] in ('CONFIRMED_REMOVED', 'CONFIRMED_ABSENT'):
            if Path(record['workspace']).exists() or Path(record['workspace']).is_symlink():
                unresolved.append(str(receipt))
        elif not recover or not collect(record, receipt):
            unresolved.append(str(receipt))
    return unresolved


def execute(args, state):
    if pending(state):
        raise RuntimeError('unfinished cleanup; run recover, then status before starting another check')
    repo = Path(args.repo).resolve(strict=True)
    uid = repo.stat().st_uid
    user = pwd.getpwuid(uid)
    if uid == 0:
        raise RuntimeError('source must belong to a development user, not root')
    if not args.command:
        raise RuntimeError('check command is required after --')
    task = state / uuid.uuid4().hex
    task.mkdir(mode=0o700)
    fd = os.open(state, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)
    workspace = Path('/tmp') / ('haa-work-check-' + task.name)
    record = {'schema': 1, 'workspace': str(workspace), 'uid': uid, 'pid': os.getpid(),
              'start': process_identity(os.getpid()), 'boot': Path('/proc/sys/kernel/random/boot_id').read_text().strip(),
              'command': args.command, 'source': str(repo), 'cleanup': 'ALLOCATING', 'command_exit': None}
    receipt = task / 'result.json'
    save(receipt, record)  # Register before creating any disposable payload.
    child = None
    interrupted = None

    def stop(signum, frame):
        nonlocal interrupted
        interrupted = signum
        if child is not None and child.poll() is None:
            os.killpg(child.pid, signal.SIGTERM)
        raise InterruptedError(f'interrupted by signal {signum}')

    previous = {sig: signal.signal(sig, stop) for sig in (signal.SIGTERM, signal.SIGINT)}
    try:
        workspace.mkdir(mode=0o700)
        os.chown(workspace, uid, user.pw_gid)
        s = workspace.stat()
        record['identity'] = [s.st_dev, s.st_ino, s.st_uid]
        record['cleanup'] = 'PENDING'
        save(receipt, record)
        source = workspace / 'source'
        source.mkdir()
        os.chown(source, uid, user.pw_gid)
        paths = subprocess.check_output(['runuser', '-u', user.pw_name, '--', 'git', '-C', str(repo), 'ls-files', '-z']).decode().split('\0')[:-1]
        source_digest = hashlib.sha256()
        for name in paths:
            src, dst = repo / name, source / name
            if not src.is_file() or src.is_symlink() or not src.resolve().is_relative_to(repo):
                raise RuntimeError(f'unsupported source member: {name}')
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(src, dst)
            dst.chmod(src.stat().st_mode & 0o777)
            os.chown(dst, uid, user.pw_gid)
            source_digest.update(name.encode() + b'\0')
            source_digest.update(str(dst.stat().st_mode & 0o777).encode() + b'\0')
            with dst.open('rb') as stream:
                source_digest.update(hashlib.file_digest(stream, 'sha256').digest())
        record['source_files'] = len(paths)
        record['source_digest'] = source_digest.hexdigest()
        save(receipt, record)
        for name in ('cache', 'tmp'):
            (workspace / name).mkdir()
        for base, directories, files in os.walk(workspace):
            os.chown(base, uid, user.pw_gid)
            for name in files:
                os.chown(Path(base) / name, uid, user.pw_gid)
        env = os.environ.copy()
        env.update(GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOWORK='off',
                   GOCACHE=str(workspace / 'cache'), TMPDIR=str(workspace / 'tmp'), PYTHONDONTWRITEBYTECODE='1')
        with (task / 'command.log').open('xb') as log:
            child = subprocess.Popen(['runuser', '-u', user.pw_name, '--', *args.command], cwd=source,
                                     env=env, stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
            record['child_pid'] = child.pid
            save(receipt, record)
            record['command_exit'] = child.wait()
    except (OSError, RuntimeError, subprocess.SubprocessError) as exc:
        record['execution_error'] = str(exc)
        record['command_exit'] = 128 + interrupted if interrupted else 1
    finally:
        for sig in previous:
            signal.signal(sig, signal.SIG_IGN)
        if child is not None and child.poll() is None:
            os.killpg(child.pid, signal.SIGTERM)
            try:
                child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                child.wait()
        clean = collect(record, receipt)
        for sig, handler in previous.items():
            signal.signal(sig, handler)
    print(json.dumps({'receipt': str(receipt), 'command_exit': record['command_exit'], 'cleanup': record['cleanup']}))
    return record['command_exit'] or (0 if clean else 1)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['run', 'status', 'recover'])
    parser.add_argument('--state', required=True, help='persistent dedicated ledger, outside /tmp and source projection')
    parser.add_argument('--repo', default=str(Path(__file__).resolve().parent.parent))
    args, command = parser.parse_known_args()
    args.command = command[1:] if command[:1] == ['--'] else command
    if os.geteuid() != 0:
        parser.error('run on the actual Linux Host with sudo for complete /proc reference checks; check command runs as source owner')
    state = Path(args.state)
    if not state.is_absolute() or state.is_symlink() or state.resolve() != state or state.is_relative_to('/tmp'):
        parser.error('state must be an absolute persistent canonical directory')
    state.mkdir(mode=0o700, parents=True, exist_ok=True)
    with (state / 'lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if args.action == 'run':
            return execute(args, state)
        unresolved = pending(state, recover=args.action == 'recover')
        print(json.dumps({'unresolved': unresolved, 'complete': not unresolved}))
        return int(bool(unresolved))


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, RuntimeError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)
