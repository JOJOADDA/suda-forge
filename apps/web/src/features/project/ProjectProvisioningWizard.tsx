import type { ChangeEvent } from 'react'

type Analysis = { classification: { primary_type: string; confidence: number }; decision: { selected_id: string; selected: { framework: string; language: string; runtime: string; package_manager: string }; reasons: string[] }; requirements: { description: string; priority: string; required: boolean }[] }
type ProvisioningRun = { id: string; status: string; progress?: number; runtime_id?: string; error?: string; steps?: { id: string; name: string; status: string; progress: number; cache_status?: string }[] }

type Props = {
  step: number
  prompt: string
  platforms: string
  analysis: Analysis | null
  manifest: Record<string, unknown> | null
  run: ProvisioningRun | null
  busy: boolean
  selectedProject: string
  onPromptChange: (event: ChangeEvent<HTMLTextAreaElement>) => void
  onPlatformsChange: (event: ChangeEvent<HTMLInputElement>) => void
  onAnalyze: () => void
  onManifest: () => void
  onProvision: () => void
  onReset: () => void
}

const stages = ['Understanding your idea', 'Planning architecture', 'Preparing Project Computer', 'Preparing tools', 'Installing dependencies', 'Preparing AI agents', 'Verifying environment', 'Ready']
const stageState = (step: number, index: number, run: ProvisioningRun | null) => {
  if (run?.status === 'BLOCKED_BY_ENVIRONMENT' || run?.status === 'BLOCKED') return index <= 2 ? 'blocked' : 'waiting'
  if (index < step - 1) return 'complete'
  if (index === step - 1) return 'active'
  return 'waiting'
}

export function ProjectProvisioningWizard(props: Props) {
  return <section className="workspace project-wizard">
    <div className="section-title"><div><span className="eyebrow">PROJECT INTELLIGENCE / PROVISIONING</span><h3>Create Project</h3></div><span>Stage {Math.min(props.step + 1, 8)} / 8</span></div>
    <div className="wizard-steps">{stages.map((label, index) => <span className={stageState(props.step, index, props.run)} key={label}><b>{String(index + 1).padStart(2, '0')}</b>{label}</span>)}</div>
    {props.step === 1 && <div className="wizard-panel"><p className="muted">Describe the product in your own words. SUDA FORGE will decide requirements, architecture, technology, and verification constraints.</p><textarea value={props.prompt} onChange={props.onPromptChange} placeholder="Describe the product without choosing language or framework" /><input value={props.platforms} onChange={props.onPlatformsChange} placeholder="Optional platform hints, e.g. web, mobile" /><button onClick={props.onAnalyze} disabled={props.busy || !props.selectedProject}>{props.busy ? 'Understanding your idea…' : 'Analyze intent'}</button></div>}
    {props.step === 2 && props.analysis && <div className="wizard-panel"><p><strong>{props.analysis.classification.primary_type}</strong> · confidence {(props.analysis.classification.confidence * 100).toFixed(0)}%</p><p>{props.analysis.requirements.length} requirements extracted. The selected architecture is recorded with reasons and alternatives.</p><button onClick={props.onManifest} disabled={props.busy}>{props.busy ? 'Planning architecture…' : 'Generate environment manifest'}</button></div>}
    {props.step === 3 && props.analysis && <div className="wizard-panel"><p><strong>{props.analysis.decision.selected_id}</strong></p><p>{props.analysis.decision.selected.language} / {props.analysis.decision.selected.framework} / {props.analysis.decision.selected.package_manager}</p><p>{props.analysis.decision.reasons.join(' · ')}</p><button onClick={props.onProvision} disabled={props.busy}>{props.busy ? 'Preparing Project Computer…' : 'Provision Project Computer'}</button></div>}
    {props.step >= 4 && <div className="wizard-panel"><p className="operation-title">{props.run?.status === 'BLOCKED_BY_ENVIRONMENT' || props.run?.status === 'BLOCKED' ? 'Environment capability required' : 'Provisioning progress'}</p><p>{props.run?.runtime_id ? `Runtime ${props.run.runtime_id}` : 'Runtime not allocated'}</p>{props.run?.progress !== undefined && <progress max="100" value={props.run.progress} />}{props.run?.error && <p className="error">{props.run.error}</p>}<div className="capabilities">{props.run?.steps?.map((item) => <span key={item.id}>{item.name}: {item.status}{item.cache_status ? ` · Cache ${item.cache_status}` : ''}</span>)}</div><p className="muted">Operations are reported from the backend. A blocked runtime is never shown as ready.</p><button className="ghost" onClick={props.onReset}>Start another analysis</button></div>}
  </section>
}
