from pathlib import Path
import fnmatch

exclude_patterns = [
    'frontend/**',
    '**/*_test.go',
    '**/*_mock.go',
    '**/*.pb.go',
    '**/*_grpc.pb.go',
    '**/mock/**',
    '**/types.go',
    '**/router.go',
    'cmd/**',
    'scripts/**',
    'docs/**',
    'assets/**',
    'migrations/**',
    'internal/gen/**',
    '**/*.sql',
    'internal/storage/s3.go',
    'internal/stack/client.go',
    'internal/stack/grpc_client.go',
    'internal/http/handlers/types.go',
]
covered = 0
total = 0

def is_excluded(path: str) -> bool:
    for pattern in exclude_patterns:
        if fnmatch.fnmatch(path, pattern):
            return True
        if pattern.startswith('**/') and fnmatch.fnmatch(path, pattern[3:]):
            return True
    return False

if __name__ == '__main__':
    for line in Path('coverage.out').read_text().splitlines():
        if line.startswith('mode:') or not line.strip():
            continue

        parts = line.split()
        if len(parts) != 3:
            continue

        fname, stmts, count = parts
        fname = fname.rsplit(':',1)[0]
        if fname.startswith('wargame/'):
            fname = fname[len('wargame/'):]
        if is_excluded(fname):
            continue

        n = int(stmts); c = int(count)
        total += n
        if c > 0:
            covered += n

    pct = covered * 100.0 / total
    print(f'codecov_style_statement_coverage={pct:.1f}% (covered={covered}, total={total})')
