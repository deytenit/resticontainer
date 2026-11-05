import json
import os
import re
import subprocess
import sys
from pathlib import Path


def lines(value):
    return [line.strip() for line in value.splitlines() if line.strip()]


def output(name, value):
    with open(os.environ['GITHUB_OUTPUT'], 'a') as stream:
        stream.write(f'{name}<<GITEA_VALUE\n{value}\nGITEA_VALUE\n')


def metadata(images, rules, sha):
    result = []
    for rule in lines(rules):
        fields = dict(part.split('=', 1) for part in rule.split(',') if '=' in part)
        if fields.get('enable', 'true').lower() == 'false':
            continue
        kind = fields.get('type')
        if kind == 'sha':
            tag = 'sha-' + sha[:7]
        elif kind == 'raw':
            tag = fields['value']
        else:
            raise ValueError('Unsupported metadata rule')
        if not re.fullmatch(r'[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}', tag):
            raise ValueError('Invalid image tag')
        result.extend(image.lower() + ':' + tag for image in lines(images))
    return result


def main():
    mode = sys.argv[1]
    if mode == 'login':
        subprocess.run(['docker', 'login', os.environ['REGISTRY'], '--username', os.environ['REGISTRY_USER'], '--password-stdin'], input=os.environ['REGISTRY_TOKEN'], text=True, check=True)
    elif mode == 'metadata':
        output('tags', '\n'.join(metadata(os.environ['IMAGES'], os.environ['TAG_RULES'], os.environ['SOURCE_SHA'])))
        output('labels', 'org.opencontainers.image.revision=' + os.environ['SOURCE_SHA'])
    elif mode == 'build':
        command = ['docker', 'buildx', 'build', '--metadata-file', '/tmp/image-metadata.json']
        for env, flag in [('TAGS', '--tag'), ('LABELS', '--label'), ('BUILD_ARGS', '--build-arg')]:
            for value in lines(os.environ.get(env, '')):
                command.extend([flag, value])
        command += ['--label', 'org.opencontainers.image.revision=' + os.environ['SOURCE_SHA'], '--label', 'com.ermnvldmr.ci.run=' + os.environ['RUN_ID']]
        if os.environ.get('PLATFORMS'):
            command += ['--platform', os.environ['PLATFORMS']]
        if os.environ.get('DOCKERFILE'):
            command += ['--file', os.environ['DOCKERFILE']]
        if os.environ.get('PUSH', 'false').lower() == 'true':
            command.append('--push')
        command.append(os.environ.get('CONTEXT', '.'))
        subprocess.run(command, check=True)
        output('digest', json.loads(Path('/tmp/image-metadata.json').read_text())['containerimage.digest'])
    else:
        raise ValueError('Unknown docker operation')


if __name__ == '__main__':
    main()
