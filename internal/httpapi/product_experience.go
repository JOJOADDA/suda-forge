package httpapi

import (
	"encoding/json"
	"net/http"
	"suda-forge/internal/constitution"
	"suda-forge/internal/designintelligence"
	"suda-forge/internal/knowledge"
	"suda-forge/internal/productexperience"
)

func (s Server) analyzeDesign(w http.ResponseWriter, r *http.Request) {
	if s.DesignIntelligence == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "design intelligence unavailable"})
		return
	}
	var in designintelligence.AnalysisInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	in.ProjectID = r.PathValue("project")
	out, err := s.DesignIntelligence.Analyze(in)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if s.DesignSystems == nil {
		s.DesignSystems = map[string]designintelligence.DesignSystem{}
	}
	s.DesignSystems[in.ProjectID] = out.DesignSystem
	if s.DesignStore != nil {
		_ = s.DesignStore.Save(r.Context(), out.DesignSystem)
	}
	writeJSON(w, http.StatusOK, out)
}
func (s Server) getDesign(w http.ResponseWriter, r *http.Request) {
	if d, ok := s.DesignSystems[r.PathValue("project")]; ok {
		writeJSON(w, http.StatusOK, d)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "design system not found"})
}
func (s Server) getKnowledge(w http.ResponseWriter, r *http.Request) {
	if s.KnowledgeStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "knowledge graph unavailable"})
		return
	}
	g, err := s.KnowledgeStore.Graph(r.PathValue("project"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, g)
}
func (s Server) upsertKnowledgeNode(w http.ResponseWriter, r *http.Request) {
	if s.KnowledgeStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "knowledge graph unavailable"})
		return
	}
	var n knowledge.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	n.ProjectID = r.PathValue("project")
	out, err := s.KnowledgeStore.UpsertNode(n)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (s Server) upsertKnowledgeEdge(w http.ResponseWriter, r *http.Request) {
	if s.KnowledgeStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "knowledge graph unavailable"})
		return
	}
	var e knowledge.Edge
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	e.ProjectID = r.PathValue("project")
	out, err := s.KnowledgeStore.UpsertEdge(e)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (s Server) analyzeImpact(w http.ResponseWriter, r *http.Request) {
	if s.ProductExperience == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product experience unavailable"})
		return
	}
	var in struct {
		RootNode knowledge.NodeID `json:"root_node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	out, err := s.ProductExperience.Impact(r.PathValue("project"), in.RootNode)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (s Server) getAgentContext(w http.ResponseWriter, r *http.Request) {
	if s.ProductExperience == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product experience unavailable"})
		return
	}
	var in productexperience.ContextRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && r.Method != "GET" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	in.ProjectID = r.PathValue("project")
	if in.AgentID == "" {
		in.AgentID = r.URL.Query().Get("agent_id")
	}
	if in.Task == "" {
		in.Task = r.URL.Query().Get("task")
	}
	out, err := s.ProductExperience.Assemble(in)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (s Server) planAutonomousLoop(w http.ResponseWriter, r *http.Request) {
	if s.ProductExperience == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "product experience unavailable"})
		return
	}
	var in struct {
		Blocked map[productexperience.LoopStage]string `json:"blocked"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	out, err := s.ProductExperience.PlanLoop(r.PathValue("project"), in.Blocked)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (s Server) evaluateGovernance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Constitution constitution.Constitution `json:"constitution"`
		Action       constitution.Action       `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	out := constitution.PolicyEvaluator{}.Evaluate(in.Constitution, in.Action)
	writeJSON(w, http.StatusOK, out)
}

func (s Server) getProjectActivity(w http.ResponseWriter, r *http.Request) {
	if s.ActivityLog == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "activity projection unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, s.ActivityLog.List(r.PathValue("project")))
}

func (s Server) projectActivityStream(w http.ResponseWriter, r *http.Request) {
	if s.Events == nil {
		http.Error(w, "event stream unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	project := r.PathValue("project")
	for event := range s.Events.Subscribe(r.Context()) {
		if event.ProjectID != "" && event.ProjectID != project {
			continue
		}
		data, _ := json.Marshal(event)
		_, _ = w.Write([]byte("event: " + event.Type + "\ndata: " + string(data) + "\n\n"))
		flusher.Flush()
	}
}

func (s Server) runVisualQA(w http.ResponseWriter, r *http.Request) {
	var in productexperience.VisualQARequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	in.ProjectID = r.PathValue("project")
	if in.ComputerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "computer_id is required"})
		return
	}
	if len(in.Viewports) == 0 {
		in.Viewports = []productexperience.Viewport{{Name: "mobile", Width: 390, Height: 844}, {Name: "tablet", Width: 1024, Height: 768}, {Name: "desktop", Width: 1440, Height: 900}}
	}
	out, err := s.VisualQA.Run(r.Context(), in)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	code := http.StatusOK
	if out.Status == productexperience.VisualQABlocked || out.Status == productexperience.VisualQAUnsupported {
		code = http.StatusUnprocessableEntity
	}
	writeJSON(w, code, out)
}

func (s Server) createConstitution(w http.ResponseWriter, r *http.Request) {
	if s.Constitutions == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "constitution service unavailable"})
		return
	}
	var c constitution.Constitution
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	c.ProjectID = r.PathValue("project")
	if c.ID == "" {
		c.ID = constitution.ID("constitution_" + c.ProjectID + "_" + string(c.AgentID))
	}
	if err := constitution.Validate(c); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	s.Constitutions[c.ProjectID+":"+string(c.AgentID)] = c
	if s.ConstitutionStore != nil {
		_ = s.ConstitutionStore.Save(r.Context(), c)
	}
	writeJSON(w, http.StatusCreated, c)
}
func (s Server) getConstitution(w http.ResponseWriter, r *http.Request) {
	if c, ok := s.Constitutions[r.PathValue("project")+":"+r.PathValue("agent")]; ok {
		writeJSON(w, http.StatusOK, c)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "constitution not found"})
}
