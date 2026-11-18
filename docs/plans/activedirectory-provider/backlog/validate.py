"""Validate the reviewable provider backlog offline, without GitHub or lab writes."""
import collections
import hashlib
import json
import pathlib
import re
import sys
from urllib.parse import unquote

ROOT = pathlib.Path(__file__).resolve().parent.parent
REQUIRED = ['Outcome', 'Implementation', 'Acceptance criteria', 'Verification', 'Dependencies',
            'Applicable product rules', 'Planning', 'Architecture decision alignment']


def require(condition, message):
    if not condition:
        raise ValueError(message)


def canonical_bytes(path):
    # Git may check text files out as CRLF on Windows; approval hashes use LF text.
    return path.read_bytes().replace(b'\r\n', b'\n')


def main():
    metadata = json.loads((ROOT / 'backlog/metadata.json').read_text(encoding='utf-8'))
    capabilities = json.loads((ROOT / 'capabilities.json').read_text(encoding='utf-8'))['operations']
    lock = json.loads((ROOT / 'backlog/review-lock.json').read_text(encoding='utf-8'))
    require(metadata['status'] == 'prepared-awaiting-exact-preview-approval', 'Unexpected publication state')
    require(metadata['publication']['approval_required'], 'Missing publication approval boundary')
    require(not metadata['publication']['issue_writes_performed'], 'Unexpected publication claim')
    require(len(capabilities) == 151, 'Expected exactly 151 SDK operations')
    names = [o['cmdlet'] for o in capabilities]
    require(len(names) == len(set(names)), 'Duplicate operation')
    require(sum(o['disposition'] == 'excluded' for o in capabilities) == 6, 'Expected six explicit exclusions')
    issues = metadata['issues']
    by_key = {i['repo'] + ':' + i['id']: i for i in issues}
    require(len(by_key) == len(issues), 'Duplicate managed identity')
    by_id = {i['id']: i for i in issues if i['id'] != 'INDEX'}
    graph = {}
    parents = 0
    ranks = {}
    for repo, proposed in metadata['repositories'].items():
        require(len({l['name'] for l in proposed['labels']}) == len(proposed['labels']), 'Duplicate label')
        for m in proposed['milestones']:
            require(m['due_on'] is None, 'Unapproved due date')
            ranks[(repo, m['id'])] = int(m['id'][1:])
    for key, item in by_key.items():
        body_path = ROOT / item['body']
        require(body_path.is_file(), f'Missing body: {key}')
        data = canonical_bytes(body_path)
        body = data.decode('utf-8')
        require(body.startswith(item['marker'] + '\n'), f'Incorrect marker: {key}')
        require(hashlib.sha256(data).hexdigest() == item['body_sha256'], f'Changed body: {key}')
        proposal = metadata['repositories'][item['repo']]
        labels = {l['name'] for l in proposal['labels']}
        require(set(item['labels']) <= labels, f'Unknown labels: {key}')
        require((item['repo'], item['milestone']) in ranks, f'Unknown milestone: {key}')
        graph[key] = item['blockers']
        if 'parent' in item:
            parents += 1
            require(item['parent'] in by_key, f'Missing parent: {key}')
            require(by_key[item['parent']]['repo'] == item['repo'], f'Wrong parent repository: {key}')
            require(ranks[(item['repo'], item['milestone'])] <= ranks[(item['repo'], by_key[item['parent']]['milestone'])], f'Parent gate precedes child: {key}')
            headings = re.findall(r'^## (.+)$', body, re.M)
            require(headings == REQUIRED, f'Wrong task headings: {key}: {headings}')
            require('Suggested PR title: `feat(' in body, f'Missing planned PR title: {key}')
            require('Raised implementation PR' in body, f'Missing completion gate: {key}')
        for blocker in item['blockers']:
            require(blocker in by_key or blocker in metadata['existing_external_blockers'], f'Unknown blocker: {key}: {blocker}')
            if blocker in by_key:
                before = by_key[blocker]
                require(ranks[(before['repo'], before['milestone'])] <= ranks[(item['repo'], item['milestone'])], f'Milestone inversion: {key}')
    visiting = set()
    visited = set()

    def visit(key):
        require(key not in visiting, f'Dependency cycle: {key}')
        if key in visited or key not in graph:
            return
        visiting.add(key)
        for blocker in graph[key]:
            visit(blocker)
        visiting.remove(key)
        visited.add(key)

    for key in graph:
        visit(key)
    expected_edges = {(k, b) for k, bs in graph.items() for b in bs}
    actual_edges = {(e['blocked'], e['blocked_by']) for e in metadata['dependencies']}
    require(len(actual_edges) == len(metadata['dependencies']), 'Duplicate dependency edge')
    require(expected_edges == actual_edges, 'Relationship map does not match issue bodies metadata')
    expected_ready = {k for k, i in by_key.items() if 'parent' in i and not i['blockers']}
    require(set(metadata['ready_frontier']) == expected_ready, 'Incorrect ready frontier')
    types = {}
    for op in capabilities:
        if op['disposition'] == 'excluded':
            require(not op['tickets'] and not op['types'], f'Excluded operation has delivery work: {op["cmdlet"]}')
        else:
            require(op['tickets'], f'Unmapped operation: {op["cmdlet"]}')
            require(all(t in by_id for t in op['tickets']), f'Unknown operation owner: {op["cmdlet"]}')
        for t in op['types']:
            k = (t['kind'], t['name'])
            require(k not in types or types[k] == t, f'Conflicting type definition: {k}')
            types[k] = t
            require(t['owner'] in op['tickets'], f'Incorrect type owner: {k}')
    # Check real local Markdown links; planned paths in code spans are not treated as existing files.
    for path in ROOT.rglob('*.md'):
        content = path.read_text(encoding='utf-8')
        for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)', content):
            if target.startswith(('https://', 'http://', 'mailto:', '#')):
                continue
            rel = unquote(target.split('#', 1)[0])
            require((path.parent / rel).resolve().is_file(), f'Broken link in {path.relative_to(ROOT)}: {target}')
    actual_files = {p.relative_to(ROOT).as_posix() for p in ROOT.rglob('*') if p.is_file() and p.name != 'review-lock.json' and '__pycache__' not in p.parts}
    require(actual_files == set(lock['files']), 'Review-lock file set differs')
    for relative, digest in lock['files'].items():
        require(hashlib.sha256(canonical_bytes(ROOT / relative)).hexdigest() == digest, f'Review content changed: {relative}')
    counts = collections.Counter(k[0] for k in types)
    print(f'PASS: {len(capabilities)} operations; 6 exclusions; {parents} tasks; {len(issues)} issue drafts; {len(actual_edges)} native blocker edges; no cycles/broken local links; all review hashes match.')
    print('Terraform types:', json.dumps(dict(sorted(counts.items()))))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError) as error:
        print(f'FAIL: {error}', file=sys.stderr)
        raise SystemExit(1)
