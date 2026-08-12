type Item = { id: string; label: string; detail: string }

const items: Item[] = [
  { id: 'create-project', label: 'Create Project', detail: 'Intent and provisioning' },
  { id: 'project-computer', label: 'Project Computer', detail: 'Runtime readiness' },
  { id: 'agents', label: 'Agents', detail: 'Sessions and events' },
  { id: 'orchestration', label: 'Orchestration', detail: 'Task graph' },
  { id: 'autonomous-loop', label: 'Autonomous Loop', detail: 'Persisted execution' },
  { id: 'verification', label: 'Verification', detail: 'Evidence and repair' },
  { id: 'ai-control', label: 'AI Control', detail: 'Runtimes and models' },
  { id: 'deployment', label: 'Deployment', detail: 'Release and health' },
  { id: 'models', label: 'Model Routing', detail: 'Policy explanation' },
  { id: 'product-intelligence', label: 'Knowledge', detail: 'Design, graph, impact' },
  { id: 'activity', label: 'Activity', detail: 'Live project events' },
]

export function WorkspaceNavigation() {
  const navigate = (id: string) => document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  return <nav className="workspace-nav" aria-label="SUDA FORGE workspaces">
    <div className="workspace-nav-heading"><span className="eyebrow">WORKSPACES</span><strong>Control plane</strong></div>
    {items.map((item) => <button type="button" key={item.id} onClick={() => navigate(item.id)}><span>{item.label}</span><small>{item.detail}</small></button>)}
  </nav>
}
