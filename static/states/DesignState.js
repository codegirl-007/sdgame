/**
 * Design State - Default canvas interaction mode
 * 
 * In this state, users can:
 * - Place components from sidebar
 * - Select/deselect components
 * - Edit component properties
 * - Delete components
 */

import { CanvasState } from './CanvasState.js';

export class DesignState extends CanvasState {
    enter(app) {
        super.enter(app);
        
        // Update UI to reflect design mode
        app.arrowToolBtn.classList.remove('active');
        app.canvas.style.cursor = this.getCursor();
        
        // Clear any connection state
        if (app.connectionStart) {
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
        }
        
        // Notify observers that connection mode is disabled
        app.connectionModeSubject.notifyConnectionModeChanged(false);
    }

    handleNodeClick(app, node, event) {
        event.stopPropagation();
        
        // Clear any selected connection when clicking a node
        if (app.selectedConnection) {
            app.selectedConnection.deselect();
            app.selectedConnection = null;
        }
        
        // Clear previous node selection and select this node
        if (app.selectedNode && app.selectedNode !== node) {
            app.nodeSelectionSubject.notifyNodeDeselected(app.selectedNode);
        }
        
        // Select the clicked node
        node.select();
        app.nodeSelectionSubject.notifyNodeSelected(node);
    }

    handleNodeDoubleClick(app, node) {
        // Show properties panel for the node
        app.propertiesPanelSubject.notifyPropertiesPanelRequested(node);
    }

    handleCanvasClick(app, event) {
        // Don't hide props panel if clicking on it
        if (!app.nodePropsPanel.contains(event.target)) {
            // Use observer to notify that properties panel should be closed
            if (app.selectedNode) {
                app.propertiesPanelSubject.notifyPropertiesPanelClosed(app.selectedNode);
            }
        }
        
        // Use base implementation for other clearing logic
        super.handleCanvasClick(app, event);
    }

    getCursor() {
        return 'default';
    }

    allowsPropertiesPanel() {
        return true;
    }

    getStateName() {
        return 'Design';
    }
}
