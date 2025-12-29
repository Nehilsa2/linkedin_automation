package persistence

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// ResumptionManager handles graceful shutdown and workflow resumption
type ResumptionManager struct {
	store            *Store
	activeWorkflows  map[int64]*WorkflowState
	shutdownHandlers []func()
}

// NewResumptionManager creates a new resumption manager
func NewResumptionManager(store *Store) *ResumptionManager {
	rm := &ResumptionManager{
		store:           store,
		activeWorkflows: make(map[int64]*WorkflowState),
	}

	// Set up signal handling for graceful shutdown
	rm.setupSignalHandling()

	return rm
}

// setupSignalHandling sets up handlers for SIGINT and SIGTERM
func (rm *ResumptionManager) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n\n⚠️ Received signal: %v\n", sig)
		fmt.Println("💾 Saving workflow state for resumption...")

		rm.PauseAllWorkflows()

		// Run any registered shutdown handlers
		for _, handler := range rm.shutdownHandlers {
			handler()
		}

		fmt.Println("✅ State saved. You can resume next time.")
		os.Exit(0)
	}()
}

// RegisterShutdownHandler adds a handler to be called on shutdown
func (rm *ResumptionManager) RegisterShutdownHandler(handler func()) {
	rm.shutdownHandlers = append(rm.shutdownHandlers, handler)
}

// StartWorkflow starts tracking a new workflow
func (rm *ResumptionManager) StartWorkflow(workflowType string, totalItems int, metadata map[string]interface{}) (*WorkflowState, error) {
	state := &WorkflowState{
		WorkflowType: workflowType,
		Status:       WorkflowStatusInProgress,
		TotalItems:   totalItems,
		Metadata:     metadata,
	}

	if err := rm.store.SaveWorkflowState(state); err != nil {
		return nil, err
	}

	rm.activeWorkflows[state.ID] = state
	return state, nil
}

// UpdateProgress updates the progress of an active workflow
func (rm *ResumptionManager) UpdateProgress(workflowID int64, currentIndex int, currentStep string) error {
	if state, ok := rm.activeWorkflows[workflowID]; ok {
		state.CurrentIndex = currentIndex
		state.CurrentStep = currentStep
	}
	return rm.store.UpdateWorkflowProgress(workflowID, currentIndex, currentStep)
}

// CompleteWorkflow marks a workflow as completed
func (rm *ResumptionManager) CompleteWorkflow(workflowID int64) error {
	delete(rm.activeWorkflows, workflowID)
	return rm.store.CompleteWorkflow(workflowID)
}

// PauseWorkflow pauses a specific workflow
func (rm *ResumptionManager) PauseWorkflow(workflowID int64) error {
	delete(rm.activeWorkflows, workflowID)
	return rm.store.PauseWorkflow(workflowID)
}

// PauseAllWorkflows pauses all active workflows
func (rm *ResumptionManager) PauseAllWorkflows() {
	for id := range rm.activeWorkflows {
		if err := rm.store.PauseWorkflow(id); err != nil {
			fmt.Printf("⚠️ Failed to pause workflow %d: %v\n", id, err)
		} else {
			fmt.Printf("⏸️ Paused workflow %d\n", id)
		}
	}
	rm.activeWorkflows = make(map[int64]*WorkflowState)
}

// GetResumableWorkflow checks for a resumable workflow of the given type
func (rm *ResumptionManager) GetResumableWorkflow(workflowType string) (*WorkflowState, error) {
	return rm.store.GetActiveWorkflow(workflowType)
}

// HasResumableWorkflows checks if there are any resumable workflows
func (rm *ResumptionManager) HasResumableWorkflows() bool {
	types := []string{WorkflowTypeSearch, WorkflowTypeConnect, WorkflowTypeMessage}
	for _, t := range types {
		state, _ := rm.store.GetActiveWorkflow(t)
		if state != nil && state.Status == WorkflowStatusPaused {
			return true
		}
	}
	return false
}

// PrintResumableWorkflows prints information about resumable workflows
func (rm *ResumptionManager) PrintResumableWorkflows() {
	fmt.Println("\n📋 Resumable Workflows:")
	fmt.Println("------------------------")

	types := []string{WorkflowTypeSearch, WorkflowTypeConnect, WorkflowTypeMessage}
	found := false

	for _, t := range types {
		state, _ := rm.store.GetActiveWorkflow(t)
		if state != nil && state.Status == WorkflowStatusPaused {
			found = true
			fmt.Printf("\n🔸 %s Workflow (ID: %d)\n", t, state.ID)
			fmt.Printf("   Progress: %d/%d\n", state.CurrentIndex, state.TotalItems)
			fmt.Printf("   Step: %s\n", state.CurrentStep)
			fmt.Printf("   Started: %s\n", state.StartedAt.Format("2006-01-02 15:04:05"))
			if state.PausedAt != nil {
				fmt.Printf("   Paused: %s\n", state.PausedAt.Format("2006-01-02 15:04:05"))
			}
		}
	}

	if !found {
		fmt.Println("No resumable workflows found.")
	}
}

// ResumeWorkflow resumes a paused workflow
func (rm *ResumptionManager) ResumeWorkflow(workflowID int64) (*WorkflowState, error) {
	// Get the workflow state
	for _, t := range []string{WorkflowTypeSearch, WorkflowTypeConnect, WorkflowTypeMessage} {
		state, err := rm.store.GetActiveWorkflow(t)
		if err != nil {
			continue
		}
		if state != nil && state.ID == workflowID {
			state.Status = WorkflowStatusInProgress
			if err := rm.store.SaveWorkflowState(state); err != nil {
				return nil, err
			}
			rm.activeWorkflows[workflowID] = state
			return state, nil
		}
	}
	return nil, fmt.Errorf("workflow %d not found or not resumable", workflowID)
}

// ClearPausedWorkflows clears all paused workflows (use with caution)
func (rm *ResumptionManager) ClearPausedWorkflows() error {
	types := []string{WorkflowTypeSearch, WorkflowTypeConnect, WorkflowTypeMessage}
	for _, t := range types {
		state, _ := rm.store.GetActiveWorkflow(t)
		if state != nil && state.Status == WorkflowStatusPaused {
			rm.store.FailWorkflow(state.ID, "manually_cleared")
		}
	}
	return nil
}
