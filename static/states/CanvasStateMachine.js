/**
 * Canvas State Machine - Manages state transitions for the canvas
 * 
 * This class coordinates state changes and ensures proper enter/exit calls.
 * It follows Nystrom's State Pattern implementation guidelines.
 */

import { DesignState } from './DesignState.js';
import { ConnectionState } from './ConnectionState.js';

export class CanvasStateMachine {
    constructor(app) {
        this.app = app;
        this.currentState = null;
        
        // Pre-create state instances for reuse
        this.states = {
            design: new DesignState(),
            connection: new ConnectionState()
        };
        
        // Start in design state
        this.changeState('design');
    }

    /**
     * Change to a new state
     * @param {string} stateName - Name of the state to change to
     */
    changeState(stateName) {
        const newState = this.states[stateName];
        
        if (!newState) {
            return;
        }
        
        if (this.currentState === newState) {
            return;
        }
        
        // Exit current state
        if (this.currentState) {
            this.currentState.exit(this.app);
        }
        
        // Enter new state
        const previousState = this.currentState;
        this.currentState = newState;
        this.currentState.enter(this.app);
        
        // Notify any listeners about state change
        this.onStateChanged(previousState, newState);
    }

    /**
     * Toggle between design and connection states
     */
    toggleConnectionMode() {
        const currentStateName = this.getCurrentStateName();
        
        if (currentStateName === 'design') {
            this.changeState('connection');
        } else {
            this.changeState('design');
        }
    }

    /**
     * Get the current state name
     */
    getCurrentStateName() {
        return this.currentState ? this.currentState.getStateName().toLowerCase() : 'none';
    }

    /**
     * Get the current state instance
     */
    getCurrentState() {
        return this.currentState;
    }

    /**
     * Check if currently in a specific state
     * @param {string} stateName 
     */
    isInState(stateName) {
        return this.getCurrentStateName() === stateName.toLowerCase();
    }

    /**
     * Delegate canvas click to current state
     */
    handleCanvasClick(event) {
        if (this.currentState) {
            this.currentState.handleCanvasClick(this.app, event);
        }
    }

    /**
     * Delegate node click to current state
     */
    handleNodeClick(node, event) {
        if (this.currentState) {
            this.currentState.handleNodeClick(this.app, node, event);
        }
    }

    /**
     * Delegate node double-click to current state
     */
    handleNodeDoubleClick(node) {
        if (this.currentState) {
            this.currentState.handleNodeDoubleClick(this.app, node);
        }
    }

    /**
     * Delegate drop event to current state
     */
    handleDrop(event) {
        if (this.currentState) {
            this.currentState.handleDrop(this.app, event);
        }
    }

    /**
     * Delegate keyboard event to current state
     */
    handleKeyDown(event) {
        if (this.currentState) {
            this.currentState.handleKeyDown(this.app, event);
        }
    }

    /**
     * Called when state changes - override for custom behavior
     */
    onStateChanged(previousState, newState) {
        // Could emit events, update analytics, etc.
        
        // Update any debug UI
        if (this.app.debugStateDisplay) {
            this.app.debugStateDisplay.textContent = `State: ${newState.getStateName()}`;
        }
    }

    /**
     * Get available states for debugging/UI
     */
    getAvailableStates() {
        return Object.keys(this.states);
    }

    /**
     * Force change to design state (safe reset)
     */
    resetToDesignState() {
        this.changeState('design');
    }
}
