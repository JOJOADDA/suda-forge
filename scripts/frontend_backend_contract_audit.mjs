import fs from 'node:fs'

const frontend = fs.readFileSync('apps/web/src/App.tsx', 'utf8')
const routes = fs.readFileSync('internal/httpapi/server.go', 'utf8')
const occurrences = [...frontend.matchAll(/`\$\{API\}([^`]+)`/g)]
const frontendPaths = occurrences.map((m) => m[1].split(/[?`]/)[0].replace(/\$\{[^}]+\}/g, '{param}')).filter((v, i, a) => a.indexOf(v) === i).sort()
const backendRoutes = [...routes.matchAll(/mux\.HandleFunc\("([A-Z]+) ([^"]+)"/g)].map((m) => `${m[1]} ${m[2]}`)
const normalize = (path) => path.replace(/\{[^}]+\}/g, '{param}')
const backendPaths = new Set(backendRoutes.map((r) => normalize(r.slice(r.indexOf(' ') + 1))))
const unresolved = []
for (const occurrence of occurrences) {
  const raw = occurrence[1].split(/[?`]/)[0]
  const path = raw.replace(/\$\{[^}]+\}/g, '{param}')
  if (path.startsWith('/health')) continue
  if (![...backendPaths].some((routePath) => routePath === normalize(path) || (routePath.includes('{param}') && normalize(path).startsWith(routePath.split('{param}')[0])))) unresolved.push(path)
}

console.log(JSON.stringify({ frontend_paths: frontendPaths, backend_route_count: backendRoutes.length, unresolved: [...new Set(unresolved)], sse_contract: frontend.includes('/activity/stream') && routes.includes('/activity/stream') }, null, 2))
if (unresolved.length) process.exitCode = 1
