import hashlib
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

from check_oci_attestations import verify


class AttestationTest(unittest.TestCase):
    def fixture(self, directory, subject=True, wrong=False, provenance=True, nested=False):
        blobs = {}

        def add(value, media='application/vnd.oci.image.manifest.v1+json'):
            data = json.dumps(value).encode()
            digest = hashlib.sha256(data).hexdigest()
            blobs['blobs/sha256/' + digest] = data
            return dict(mediaType=media, digest='sha256:' + digest, size=len(data))

        image = add(dict(config=add(dict(os='linux', architecture='amd64')), layers=[]))
        digest = '0' * 64 if wrong else image['digest'].split(':', 1)[1]
        layers = []
        types = ['https://spdx.dev/Document']
        if provenance:
            types.append('https://slsa.dev/provenance/v0.2')
        for predicate in types:
            layers.append(add(dict(predicateType=predicate, subject=[dict(name='fixture', digest=dict(sha256=digest))] if subject else []), 'application/vnd.in-toto+json'))
        attestation = add(dict(config=add({}), layers=layers))
        index = dict(manifests=[image, attestation])
        if nested:
            index = dict(manifests=[add(index, 'application/vnd.oci.image.index.v1+json')])
        blobs['index.json'] = json.dumps(index).encode()
        path = Path(directory) / 'image.tar'
        with tarfile.open(path, 'w') as archive:
            for name, data in blobs.items():
                info = tarfile.TarInfo(name)
                info.size = len(data)
                archive.addfile(info, io.BytesIO(data))
        return path

    def test_named_image_and_nested_index(self):
        with tempfile.TemporaryDirectory() as directory:
            verify(self.fixture(directory, nested=True), 'amd64')

    def test_rejects_unbound_or_incomplete_proofs(self):
        for options in [dict(subject=False), dict(wrong=True), dict(provenance=False)]:
            with self.subTest(options=options), tempfile.TemporaryDirectory() as directory:
                with self.assertRaises(ValueError):
                    verify(self.fixture(directory, **options), 'amd64')

    def test_rejects_wrong_architecture(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(ValueError):
                verify(self.fixture(directory), 'arm64')
