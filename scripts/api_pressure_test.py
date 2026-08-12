import concurrent.futures
import json
import statistics
import sys
import time
from collections import Counter
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

BASE = sys.argv[1] if len(sys.argv) > 1 else 'http://127.0.0.1:18080'
PROJECT = sys.argv[2] if len(sys.argv) > 2 else f'pressure-{int(time.time())}'
COUNT = int(sys.argv[3]) if len(sys.argv) > 3 else 32
WORKERS = int(sys.argv[4]) if len(sys.argv) > 4 else 8


def request(method, path, body=None, timeout=8, raw=False):
    data = body if raw else (None if body is None else json.dumps(body).encode())
    headers = {'Content-Type': 'application/json'} if body is not None else {}
    started = time.perf_counter()
    try:
        response = urlopen(Request(BASE + path, method=method, data=data, headers=headers), timeout=timeout)
        payload = response.read(4096)
        status = response.status
        error = ''
    except HTTPError as exc:
        payload = exc.read(4096)
        status = exc.code
        error = payload.decode(errors='replace')[:240]
    except (URLError, TimeoutError, OSError) as exc:
        status = 0
        error = repr(exc)
    return {'status': status, 'ms': (time.perf_counter() - started) * 1000, 'error': error}


def run(label, method, path, body, count, workers, raw=False):
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        results = list(pool.map(lambda _: request(method, path, body, raw=raw), range(count)))
    statuses = Counter(item['status'] for item in results)
    latencies = sorted(item['ms'] for item in results)
    errors = [item['error'] for item in results if item['error']]
    percentile = lambda p: latencies[min(len(latencies) - 1, int(len(latencies) * p / 100))]
    return {
        'label': label,
        'count': count,
        'workers': workers,
        'statuses': dict(statuses),
        'errors': len(errors),
        'sample_errors': errors[:3],
        'latency_ms': {'min': round(latencies[0], 2), 'median': round(statistics.median(latencies), 2), 'p95': round(percentile(95), 2), 'max': round(latencies[-1], 2)},
    }


def main():
    loop_body = {'goal': 'pressure-test persisted loop', 'blocked': {}}
    verification_body = {'project_id': PROJECT, 'runtime_id': 'pressure-runtime', 'workspace': '/workspace', 'profile': 'standard', 'policy': 'STANDARD', 'checks': [{'id': 'build', 'type': 'BUILD', 'name': 'Build', 'required': True, 'timeout': 2000000000, 'configuration': {'argv': ['true']}}]}
    malformed = '{not-json'
    reports = []
    reports.append(run('loop_start_1x', 'POST', f'/api/projects/{PROJECT}/autonomous-loop/start', loop_body, 1, 1))
    reports.append(run(f'loop_status_{COUNT}c', 'GET', f'/api/projects/{PROJECT}/autonomous-loop/loop_{PROJECT}', None, COUNT, WORKERS))
    reports.append(run(f'loop_duplicate_start_{COUNT}c', 'POST', f'/api/projects/{PROJECT}/autonomous-loop/start', loop_body, COUNT, WORKERS))
    reports.append(run(f'loop_malformed_{COUNT}c', 'POST', f'/api/projects/{PROJECT}/autonomous-loop/start', b'{not-json', COUNT, WORKERS, raw=True))
    reports.append(run(f'verification_valid_{COUNT}c', 'POST', '/api/verifications', verification_body, COUNT, WORKERS))
    reports.append(run(f'verification_malformed_{COUNT}c', 'POST', '/api/verifications', b'{not-json', COUNT, WORKERS, raw=True))
    reports.append(run(f'verification_missing_id_{COUNT}c', 'POST', '/api/verifications', {'checks': []}, COUNT, WORKERS))
    reports.append(run(f'loop_resume_{COUNT}c', 'POST', f'/api/projects/{PROJECT}/autonomous-loop/loop_{PROJECT}/resume', {}, COUNT, WORKERS))
    reports.append(run(f'loop_unknown_resume_{COUNT}c', 'POST', f'/api/projects/{PROJECT}/autonomous-loop/unknown/resume', {}, COUNT, WORKERS))
    print(json.dumps({'base': BASE, 'project': PROJECT, 'reports': reports}, indent=2))


if __name__ == '__main__':
    main()
