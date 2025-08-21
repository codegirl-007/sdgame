/**
 * Connection State - Arrow mode for connecting components
 * 
 * In this state, users can:
 * - Click nodes to start/end connections
 * - See visual feedback for connection process
 * - Cannot edit properties (properties panel disabled)
 */

import { CanvasState } from './CanvasState.js';
import { Connection } from '../connection.js';

export class ConnectionState extends CanvasState {
    enter(app) {
        super.enter(app);
        
        // Update UI to reflect connection mode
        app.arrowToolBtn.classList.add('active');
        app.canvas.style.cursor = this.getCursor();
        
        // Hide properties panel (connection mode disables editing)
        if (app.selectedNode) {
            app.propertiesPanelSubject.notifyPropertiesPanelClosed(app.selectedNode);
        }
        
        // Notify observers that connection mode is enabled
        app.connectionModeSubject.notifyConnectionModeChanged(true);
    }

    exit(app) {
        super.exit(app);
        
        // Clear any pending connection
        if (app.connectionStart) {
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
        }
    }

    handleNodeClick(app, node, event) {
        event.stopPropagation();
        
        // Clear any selected connection when starting a new connection
        if (app.selectedConnection) {
            app.selectedConnection.deselect();
            app.selectedConnection = null;
        }
        
        if (!app.connectionStart) {
            // First click - start connection
            app.connectionStart = node;
            node.group.classList.add('selected');
            console.log('Connection started from:', node.type);
            
        } else if (app.connectionStart === node) {
            // Clicked same node - cancel connection
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
            console.log('Connection cancelled');
            
        } else {
            // Second click - complete connection
            this.createConnection(app, app.connectionStart, node);
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
        }
    }

    handleNodeDoubleClick(app, node) {
        // In connection mode, double-click does nothing
        // Properties panel is disabled in this state
        console.log('Properties panel disabled in connection mode');
    }

    handleCanvasClick(app, event) {
        // Cancel any pending connection when clicking canvas
        if (app.connectionStart) {
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
            console.log('Connection cancelled by canvas click');
        }
        
        // Don't clear node selections in connection mode
        // Users should be able to see what's selected while connecting
    }

    /**
     * Create a connection between two nodes
     * @param {CanvasApp} app 
     * @param {ComponentNode} startNode 
     * @param {ComponentNode} endNode 
     */
    createConnection(app, startNode, endNode) {
        // Set up pending connection for modal
        app.pendingConnection = { start: startNode, end: endNode };
        
        // Setup connection modal (reuse existing modal logic)
        Connection.setupModal(app);
        Connection.labelInput.value = 'Read traffic';
        Connection.protocolInput.value = 'HTTP';
        Connection.tlsCheckbox.checked = false;
        Connection.capacityInput.value = '1000';
        Connection.modal.style.display = 'block';
        
        console.log(`Connection setup: ${startNode.type} -> ${endNode.type}`);
    }

    getCursor() {
        return 'crosshair';
    }

    allowsPropertiesPanel() {
        return false; // Disable properties panel in connection mode
    }

    getStateName() {
        return 'Connection';
    }
}
