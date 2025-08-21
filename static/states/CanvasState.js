/**
 * Base Canvas State - Nystrom's State Pattern Implementation
 * 
 * This abstract base class defines the interface that all canvas states must implement.
 * Each state handles user interactions differently, eliminating the need for mode checking.
 */

export class CanvasState {
    /**
     * Called when entering this state
     * @param {CanvasApp} app - The canvas application context
     */
    enter(app) {
        // Override in concrete states
    }

    /**
     * Called when exiting this state  
     * @param {CanvasApp} app - The canvas application context
     */
    exit(app) {
        // Override in concrete states
    }

    /**
     * Handle clicks on the canvas background
     * @param {CanvasApp} app - The canvas application context
     * @param {MouseEvent} event - The click event
     */
    handleCanvasClick(app, event) {
        // Default: clear selections
        if (event.detail > 1) return; // Ignore double-clicks
        
        // Clear any connection start
        if (app.connectionStart) {
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
        }
        
        // Clear node selection via observer
        if (app.selectedNode) {
            app.nodeSelectionSubject.notifyNodeDeselected(app.selectedNode);
            app.selectedNode = null;
        }
        
        // Clear connection selection
        if (app.selectedConnection) {
            app.selectedConnection.deselect();
            app.selectedConnection = null;
        }
    }

    /**
     * Handle single clicks on nodes
     * @param {CanvasApp} app - The canvas application context  
     * @param {ComponentNode} node - The clicked node
     * @param {MouseEvent} event - The click event
     */
    handleNodeClick(app, node, event) {
        // Override in concrete states
        throw new Error(`${this.constructor.name} must implement handleNodeClick()`);
    }

    /**
     * Handle double clicks on nodes
     * @param {CanvasApp} app - The canvas application context
     * @param {ComponentNode} node - The double-clicked node  
     */
    handleNodeDoubleClick(app, node) {
        // Override in concrete states
        throw new Error(`${this.constructor.name} must implement handleNodeDoubleClick()`);
    }

    /**
     * Handle component drops from sidebar
     * @param {CanvasApp} app - The canvas application context
     * @param {DragEvent} event - The drop event
     */
    async handleDrop(app, event) {
        // Default implementation - most states allow dropping
        const type = event.dataTransfer.getData('text/plain');
        
        // Import PluginRegistry dynamically to avoid circular imports
        const { PluginRegistry } = await import('../pluginRegistry.js');
        const plugin = PluginRegistry.get(type);
        if (!plugin) return;
        
        const pt = app.canvas.createSVGPoint();
        pt.x = event.clientX;
        pt.y = event.clientY;
        
        const svgP = pt.matrixTransform(app.canvas.getScreenCTM().inverse());
        const x = svgP.x - app.componentSize.width / 2;
        const y = svgP.y - app.componentSize.height / 2;
        
        const { generateDefaultProps } = await import('../utils.js');
        const { ComponentNode } = await import('../node.js');
        
        const props = generateDefaultProps(plugin);
        const node = new ComponentNode(type, x, y, app, props);
        node.x = x;
        node.y = y;
    }

    /**
     * Handle keyboard events
     * @param {CanvasApp} app - The canvas application context
     * @param {KeyboardEvent} event - The keyboard event
     */
    handleKeyDown(app, event) {
        // Default: handle delete key
        if (event.key === 'Backspace' || event.key === 'Delete') {
            if (app.selectedConnection) {
                app.canvas.removeChild(app.selectedConnection.line);
                app.canvas.removeChild(app.selectedConnection.text);
                const index = app.connections.indexOf(app.selectedConnection);
                if (index !== -1) app.connections.splice(index, 1);
                app.selectedConnection = null;
            } else if (app.selectedNode) {
                app.canvas.removeChild(app.selectedNode.group);
                app.placedComponents = app.placedComponents.filter(n => n !== app.selectedNode);
                app.connections = app.connections.filter(conn => {
                    if (conn.start === app.selectedNode || conn.end === app.selectedNode) {
                        app.canvas.removeChild(conn.line);
                        app.canvas.removeChild(conn.text);
                        return false;
                    }
                    return true;
                });
                app.selectedNode = null;
                app.activeNode = null;
            }
        }
    }

    /**
     * Get the display name of this state
     */
    getStateName() {
        return this.constructor.name.replace('State', '');
    }

    /**
     * Get the cursor style for this state
     */
    getCursor() {
        return 'default';
    }

    /**
     * Whether this state allows properties panel to open
     */
    allowsPropertiesPanel() {
        return true;
    }
}
