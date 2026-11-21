#!/usr/bin/env python3
"""Build a local Docker archive from the pinned Python image and its own .pyc files."""

import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile


CACHE_RECIPE = r'''
import compileall
from pathlib import Path
from py_compile import PycInvalidationMode
import sys
import tarfile

source = Path('/usr/local/lib/python3.14')
sys.pycache_prefix = '/tmp/compiled'
if not compileall.compile_dir(source, quiet=2, workers=1, invalidation_mode=PycInvalidationMode.TIMESTAMP):
    raise SystemExit('Python compilation failed')
result = Path(sys.pycache_prefix) / str(source).lstrip('/')
paths = sorted(result.rglob('*.pyc'))
if not paths or any(not path.is_file() or path.is_symlink() for path in paths):
    raise SystemExit('Python cache has unexpected members')
with tarfile.open(fileobj=sys.stdout.buffer, mode='w|', format=tarfile.USTAR_FORMAT) as archive:
    directories = sorted({path.parent.relative_to(result) for path in paths})
    for directory in directories:
        entry = tarfile.TarInfo(str(source.relative_to('/') / directory / '__pycache__'))
        entry.type = tarfile.DIRTYPE
        entry.mode = 0o755
        archive.addfile(entry)
    for path in paths:
        directory = source.relative_to('/') / path.parent.relative_to(result)
        entry = tarfile.TarInfo(str(directory / '__pycache__' / path.name))
        entry.mode = 0o644
        entry.size = path.stat().st_size
        with path.open('rb') as stream:
            archive.addfile(entry, stream)
'''


def docker(*args, stdout=subprocess.PIPE):
    env = {'PATH': '/usr/bin:/bin', 'DOCKER_HOST': 'unix:///var/run/docker.sock'}
    return subprocess.run(['/usr/bin/docker', *args], check=True, stdout=stdout, timeout=180, env=env).stdout


def digest(value):
    return 'sha256:' + hashlib.sha256(value).hexdigest()


def json_bytes(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def read_member(archive, name, limit):
    entry = archive.getmember(name)
    if not entry.isfile() or not 0 < entry.size <= limit:
        raise ValueError('invalid base image metadata')
    with archive.extractfile(entry) as stream:
        return stream.read()


def write_member(archive, name, content):
    header = tarfile.TarInfo(name)
    header.size = len(content)
    header.mode = 0o644
    archive.addfile(header, io.BytesIO(content))


def materialize(lock, output, identity_only):
    base = lock['python_image']['reference']
    if not re.fullmatch(r'python:3\.14\.\d+-slim-bookworm@sha256:[a-f0-9]{64}', base):
        raise ValueError('expected a pinned official Python 3.14 image')
    with tempfile.TemporaryDirectory(prefix='python-resolver-build-') as scratch:
        root = Path(scratch)
        image_tar = root / 'official.tar'
        compiler_id = root / 'compiler.cid'
        base_config = docker('image', 'inspect', '--format', '{{.Id}} {{.Os}}/{{.Architecture}}', base).decode().split()
        if len(base_config) != 2 or not re.fullmatch(r'sha256:[a-f0-9]{64}', base_config[0]) or base_config[1] != 'linux/amd64':
            raise ValueError('invalid official image config identity')
        try:
            layer = docker('run', '--rm', '--cidfile', str(compiler_id), '--pull', 'never', '--runtime', 'runc',
                           '--network', 'none', '--read-only', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges',
                           '--memory', '512m', '--cpus', '1', '--pids-limit', '64',
                           '--tmpfs', '/tmp:rw,noexec,nosuid,nodev,size=64m', '--entrypoint', '/usr/local/bin/python3',
                           base, '-I', '-B', '-c', CACHE_RECIPE)
        finally:
            if compiler_id.is_file():
                owned = compiler_id.read_text().strip()
                if not re.fullmatch('[a-f0-9]{64}', owned):
                    raise ValueError('unexpected compiler container identity')
                active = docker('ps', '-aq', '--no-trunc', '--filter', 'id='+owned).decode().split()
                if active:
                    if active != [owned]:
                        raise ValueError('unexpected compiler container identity')
                    docker('rm', '--force', owned)
        if not 0 < len(layer) <= 32 << 20:
            raise ValueError('compiled Python cache exceeds its bound')
        docker('image', 'save', '--platform', 'linux/amd64', '--output', str(image_tar), base)
        with tarfile.open(image_tar, 'r:') as official:
            [manifest] = json.loads(read_member(official, 'manifest.json', 1 << 20))
            original = read_member(official, manifest['Config'], 1 << 20)
            if digest(original) != base_config[0]:
                raise ValueError('official image config differs from the base digest')
            config = json.loads(original)
            diff_id = digest(layer)
            config['rootfs']['diff_ids'].append(diff_id)
            config['history'].append({'created_by': 'Heliopause reproducible Python bytecode cache'})
            derived = json_bytes(config)
            identity = {'base_config': base_config[0], 'layer_diff_id': diff_id, 'config': digest(derived)}
            if identity_only:
                print(json.dumps(identity, sort_keys=True))
                return
            if identity != lock['python_resolver_image']:
                raise ValueError('derived Python image differs from the lock')
            filename = identity['config'].removeprefix('sha256:') + '.json'
            layer_name = diff_id.removeprefix('sha256:') + '/layer.tar'
            with tempfile.NamedTemporaryFile(dir=output.parent, prefix='.python-resolver-') as staged:
                with tarfile.open(fileobj=staged, mode='w') as target:
                    for name in manifest['Layers']:
                        entry = official.getmember(name)
                        if not entry.isfile():
                            raise ValueError('invalid base image layer')
                        with official.extractfile(entry) as stream:
                            target.addfile(entry, stream)
                    write_member(target, filename, derived)
                    write_member(target, layer_name, layer)
                    record = [{'Config': filename, 'RepoTags': None, 'Layers': manifest['Layers'] + [layer_name]}]
                    write_member(target, 'manifest.json', json_bytes(record))
                staged.flush()
                os.fsync(staged.fileno())
                os.link(staged.name, output)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--runtime-lock', type=Path, required=True)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--output', type=Path)
    group.add_argument('--identity-only', action='store_true')
    args = parser.parse_args()
    materialize(json.loads(args.runtime_lock.read_text()), args.output, args.identity_only)


if __name__ == '__main__':
    main()
