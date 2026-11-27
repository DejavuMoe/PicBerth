import hashlib
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

from check_oci_attestations import verify
from publish_oci_image import publish, sha256


class AttestationTest(unittest.TestCase):
    def fixture(self, directory, subject=True, wrong=False, provenance=True, nested=False, arch='amd64', reference=True):
        blobs = {}

        def add(value, media='application/vnd.oci.image.manifest.v1+json'):
            data = json.dumps(value).encode()
            digest = hashlib.sha256(data).hexdigest()
            blobs['blobs/sha256/' + digest] = data
            return dict(mediaType=media, digest='sha256:' + digest, size=len(data))

        image = add(dict(config=add(dict(os='linux', architecture=arch)), layers=[]))
        image['platform'] = dict(os='linux', architecture=arch)
        digest = '0' * 64 if wrong else image['digest'].split(':', 1)[1]
        layers = []
        types = ['https://spdx.dev/Document']
        if provenance:
            types.append('https://slsa.dev/provenance/v0.2')
        for predicate in types:
            layers.append(add(dict(predicateType=predicate, subject=[dict(name='fixture', digest=dict(sha256=digest))] if subject else []), 'application/vnd.in-toto+json'))
        attestation = add(dict(config=add({}), layers=layers))
        if reference:
            attestation['annotations'] = {'vnd.docker.reference.type': 'attestation-manifest',
                                          'vnd.docker.reference.digest': image['digest']}
        index = dict(manifests=[image, attestation])
        if nested:
            index = dict(manifests=[add(index, 'application/vnd.oci.image.index.v1+json')])
        blobs['index.json'] = json.dumps(index).encode()
        path = Path(directory) / (arch + '.tar')
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

    def registry(self, directory, arm_options):
        tags, contents, writes = {}, {}, []
        for arch, options in [('amd64', {}), ('arm64', arm_options)]:
            with tarfile.open(self.fixture(directory, arch=arch, **options)) as archive:
                index = archive.extractfile('index.json').read()
                contents[sha256(index)] = index
                tags[f'registry.example/picberth:staged-{arch}'] = sha256(index)
                for member in archive.getmembers():
                    if member.name.startswith('blobs/sha256/'):
                        contents[member.name.split('/', 1)[1].replace('/', ':', 1)] = archive.extractfile(member).read()

        def run(group, command, reference, *options, data=None):
            match group, command:
                case 'image', 'digest':
                    return tags[reference].encode()
                case 'tag', 'ls':
                    names = [tag.split(':', 1)[1] for tag in tags if tag.startswith(reference + ':')]
                    return '\n'.join(names).encode()
                case 'manifest', 'get':
                    return contents[reference.split('@', 1)[1]]
                case 'blob', 'get':
                    return contents[options[0]]
                case 'manifest', 'put':
                    writes.append(json.loads(data))
                    tags[reference] = sha256(data)
                    return b''
                case _:
                    raise AssertionError((group, command, reference))

        return tags, writes, run

    def test_publish_validates_both_architectures_before_writing(self):
        for arm_options in ({}, dict(provenance=False), dict(wrong=True), dict(reference=False)):
            with self.subTest(arm_options=arm_options), tempfile.TemporaryDirectory() as directory:
                tags, writes, run = self.registry(directory, arm_options)
                if arm_options:
                    with self.assertRaises(ValueError):
                        publish('staged', 'registry.example/picberth:release', run=run)
                    self.assertEqual(writes, [])
                else:
                    publish('staged', 'registry.example/picberth:release', run=run)
                    self.assertEqual(len(writes), 1)
                    self.assertEqual([d['platform']['architecture'] for d in writes[0]['manifests']
                                      if 'platform' in d], ['amd64', 'arm64'])

    def test_publish_never_replaces_an_existing_release_tag(self):
        target = 'registry.example/picberth:release'
        with tempfile.TemporaryDirectory() as directory:
            tags, writes, run = self.registry(directory, {})
            tags[target] = 'sha256:' + '1' * 64
            with self.assertRaises(ValueError):
                publish('staged', target, run=run)
            self.assertEqual(writes, [])
            self.assertEqual(tags[target], 'sha256:' + '1' * 64)

    def test_publish_accepts_a_release_tag_it_already_wrote(self):
        target = 'registry.example/picberth:release'
        with tempfile.TemporaryDirectory() as directory:
            tags, writes, run = self.registry(directory, {})
            publish('staged', target, run=run)
            published = tags[target]
            publish('staged', target, run=run)
            self.assertEqual(len(writes), 1)
            self.assertEqual(tags[target], published)
