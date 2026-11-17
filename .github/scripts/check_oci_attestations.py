import hashlib
import json
import sys
import tarfile


def verify(path, architecture):
    images, statements, seen = {}, [], set()
    with tarfile.open(path) as archive:
        def load(descriptor):
            algorithm, digest = descriptor['digest'].split(':', 1)
            if algorithm != 'sha256':
                raise ValueError('Unsupported OCI digest algorithm')
            data = archive.extractfile('blobs/sha256/' + digest).read()
            if len(data) != descriptor['size'] or hashlib.sha256(data).hexdigest() != digest:
                raise ValueError('OCI blob digest mismatch')
            return json.loads(data)

        def visit(descriptor):
            if descriptor['digest'] in seen:
                return
            seen.add(descriptor['digest'])
            manifest = load(descriptor)
            if 'manifests' in manifest:
                for child in manifest['manifests']:
                    visit(child)
                return
            config = load(manifest['config'])
            if config.get('os') == 'linux':
                images[descriptor['digest'].split(':', 1)[1]] = config.get('architecture')
            for layer in manifest.get('layers', []):
                if layer['mediaType'] == 'application/vnd.in-toto+json':
                    statements.append(load(layer))

        for descriptor in json.load(archive.extractfile('index.json'))['manifests']:
            visit(descriptor)
    if len(images) != 1 or set(images.values()) != {architecture}:
        raise ValueError('Unexpected runnable image architecture')
    predicates = set()
    for statement in statements:
        if not any(subject.get('digest', {}).get('sha256') in images for subject in statement.get('subject', [])):
            raise ValueError('Attestation subject does not bind the runnable image')
        predicates.add(statement['predicateType'])
    if 'https://spdx.dev/Document' not in predicates or not any(p.startswith('https://slsa.dev/provenance/') for p in predicates):
        raise ValueError('Both an SBOM and provenance are required')


if __name__ == '__main__':
    if len(sys.argv) != 3:
        sys.exit('Usage: check_oci_attestations.py IMAGE.tar ARCH')
    verify(sys.argv[1], sys.argv[2])
    print('Verified OCI SBOM and provenance subjects for ' + sys.argv[2])
