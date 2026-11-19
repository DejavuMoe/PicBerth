import hashlib
import json
import sys
import tarfile


def verify_index(index, read, architecture, *, require_references=False):
    images, statements, seen = {}, [], set()

    def load(descriptor, kind):
        algorithm, digest = descriptor['digest'].split(':', 1)
        if algorithm != 'sha256' or len(digest) != 64 or any(c not in '0123456789abcdef' for c in digest):
            raise ValueError('Unsupported OCI digest')
        data = read(kind, descriptor['digest'])
        if len(data) != descriptor['size'] or hashlib.sha256(data).hexdigest() != digest:
            raise ValueError('OCI blob digest mismatch')
        return json.loads(data)

    def visit(descriptor):
        if descriptor['digest'] in seen:
            return
        seen.add(descriptor['digest'])
        manifest = load(descriptor, 'manifest')
        if 'manifests' in manifest:
            for child in manifest['manifests']:
                visit(child)
            return
        config = load(manifest['config'], 'blob')
        if config.get('os') == 'linux':
            images[descriptor['digest'].split(':', 1)[1]] = config.get('architecture')
        annotations = descriptor.get('annotations', {})
        reference = manifest.get('subject', {}).get('digest')
        if annotations.get('vnd.docker.reference.type') == 'attestation-manifest':
            reference = annotations.get('vnd.docker.reference.digest')
        for layer in manifest.get('layers', []):
            if layer['mediaType'] == 'application/vnd.in-toto+json':
                statements.append((load(layer, 'blob'), reference))

    for descriptor in index['manifests']:
        visit(descriptor)
    if len(images) != 1 or set(images.values()) != {architecture}:
        raise ValueError('Unexpected runnable image architecture')
    predicates = set()
    runnable = 'sha256:' + next(iter(images))
    for statement, reference in statements:
        if require_references and reference != runnable:
            raise ValueError('Attestation descriptor does not refer to the runnable image')
        if not any(subject.get('digest', {}).get('sha256') in images for subject in statement.get('subject', [])):
            raise ValueError('Attestation subject does not bind the runnable image')
        predicates.add(statement['predicateType'])
    if 'https://spdx.dev/Document' not in predicates or not any(p.startswith('https://slsa.dev/provenance/') for p in predicates):
        raise ValueError('Both an SBOM and provenance are required')
    return runnable


def verify(path, architecture):
    with tarfile.open(path) as archive:
        def read(_kind, digest):
            with archive.extractfile('blobs/' + digest.replace(':', '/', 1)) as blob:
                return blob.read()

        verify_index(json.load(archive.extractfile('index.json')), read, architecture)


if __name__ == '__main__':
    if len(sys.argv) != 3:
        sys.exit('Usage: check_oci_attestations.py IMAGE.tar ARCH')
    verify(sys.argv[1], sys.argv[2])
    print('Verified OCI SBOM and provenance subjects for ' + sys.argv[2])
