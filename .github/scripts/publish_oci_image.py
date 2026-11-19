"""Verify both staged images before publishing their shared release tag."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys

from check_oci_attestations import verify_index


def sha256(data):
    return 'sha256:' + hashlib.sha256(data).hexdigest()


def regctl(*args, data=None):
    config = Path.home() / '.docker' / 'config.json'
    return subprocess.run([
        'docker', 'run', '--rm', '-i', '-v', f'{config}:/home/appuser/.docker/config.json:ro',
        'regclient/regctl:v0.11.7@sha256:45b57933c2b67bb1e280b4baaa220ef1ab212c81268ab19589fa1e83719c7c82',
        *args,
    ], input=data, stdout=subprocess.PIPE, check=True).stdout


def publish(source, target, run=regctl):
    repository = target.rsplit(':', 1)[0]

    def manifest(digest):
        return run('manifest', 'get', f'{repository}@{digest}', '--format', 'raw-body')

    def read(kind, digest):
        if kind == 'manifest':
            return manifest(digest)
        return run('blob', 'get', repository, digest)

    indexes = []
    for arch in ('amd64', 'arm64'):
        digest = run('image', 'digest', f'{repository}:{source}-{arch}').decode().strip()
        raw_index = manifest(digest)
        if sha256(raw_index) != digest:
            raise ValueError('Staged image index digest mismatch')
        index = json.loads(raw_index)
        runnable = verify_index(index, read, arch, require_references=True)
        platforms = [descriptor for descriptor in index['manifests']
                     if descriptor.get('platform', {}).get('os') == 'linux']
        if len(platforms) != 1 or platforms[0]['platform'].get('architecture') != arch or platforms[0]['digest'] != runnable:
            raise ValueError(f'Unexpected Linux platform descriptor for {arch}')
        indexes.append(index)
        print(f'Verified {arch} index {digest}', flush=True)

    combined = indexes[0]
    combined['manifests'].extend(indexes[1]['manifests'])
    body = json.dumps(combined, separators=(',', ':')).encode()
    run('manifest', 'put', target, data=body)
    if run('image', 'digest', target).decode().strip() != sha256(body):
        raise ValueError('Published image index digest mismatch')


if __name__ == '__main__':
    if len(sys.argv) != 3:
        sys.exit('Usage: publish_oci_image.py SOURCE_TAG_PREFIX REPOSITORY:RELEASE_TAG')
    publish(sys.argv[1], sys.argv[2])
