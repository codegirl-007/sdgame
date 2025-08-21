/**
 * Dedicated Observer Pattern Implementation
 * 
 * Each observer is dedicated to a particular concern and is type-safe.
 * This provides clean separation of concerns and maintainable event handling.
 */

import { PluginRegistry } from './pluginRegistry.js';

/**
 * NodeSelectionObserver - Dedicated to node selection events only
 */
export class NodeSelectionObserver {
    constructor() {
        this.observers = [];
    }

    // Add a specific observer for node selection changes
    addObserver(observer) {
        if (typeof observer.onNodeSelected !== 'function' ||
            typeof observer.onNodeDeselected !== 'function') {
            throw new Error('Observer must implement onNodeSelected and onNodeDeselected methods');
        }
        this.observers.push(observer);
    }

    removeObserver(observer) {
        const index = this.observers.indexOf(observer);
        if (index !== -1) {
            this.observers.splice(index, 1);
        }
    }

    // Notify all observers about node selection
    notifyNodeSelected(node) {
        for (const observer of this.observers) {
            observer.onNodeSelected(node);
        }
    }

    // Notify all observers about node deselection  
    notifyNodeDeselected(node) {
        for (const observer of this.observers) {
            observer.onNodeDeselected(node);
        }
    }
}

/**
 * PropertiesPanelObserver - Dedicated to properties panel events only
 */
export class PropertiesPanelObserver {
    constructor() {
        this.observers = [];
    }

    addObserver(observer) {
        if (typeof observer.onPropertiesPanelRequested !== 'function') {
            throw new Error('Observer must implement onPropertiesPanelRequested method');
        }
        this.observers.push(observer);
    }

    removeObserver(observer) {
        const index = this.observers.indexOf(observer);
        if (index !== -1) {
            this.observers.splice(index, 1);
        }
    }

    notifyPropertiesPanelRequested(node) {
        for (const observer of this.observers) {
            observer.onPropertiesPanelRequested(node);
        }
    }

    notifyPropertiesPanelClosed(node) {
        for (const observer of this.observers) {
            if (observer.onPropertiesPanelClosed) {
                observer.onPropertiesPanelClosed(node);
            }
        }
    }
}

/**
 * ConnectionModeObserver - Dedicated to connection/arrow mode events
 */
export class ConnectionModeObserver {
    constructor() {
        this.observers = [];
    }

    addObserver(observer) {
        if (typeof observer.onConnectionModeChanged !== 'function') {
            throw new Error('Observer must implement onConnectionModeChanged method');
        }
        this.observers.push(observer);
    }

    removeObserver(observer) {
        const index = this.observers.indexOf(observer);
        if (index !== -1) {
            this.observers.splice(index, 1);
        }
    }

    notifyConnectionModeChanged(isEnabled) {
        for (const observer of this.observers) {
            observer.onConnectionModeChanged(isEnabled);
        }
    }
}

/**
 * Properties Panel Manager - Implements observer interfaces
 */
export class PropertiesPanelManager {
    constructor(panelElement, saveButton) {
        this.panel = panelElement;
        this.saveButton = saveButton;
        this.activeNode = null;

        this.setupDOMEventListeners();
    }

    // Implement the observer interface for properties panel events
    onPropertiesPanelRequested(node) {
        const plugin = PluginRegistry.get(node.type);
        
        if (!plugin) {
            this.hidePanel();
            return;
        }
        
        this.showPanel(node, plugin);
    }

    onPropertiesPanelClosed(node) {
        if (this.activeNode === node) {
            this.hidePanel();
        }
    }

    // Implement the observer interface for connection mode events  
    onConnectionModeChanged(isEnabled) {
        if (isEnabled && this.activeNode) {
            this.hidePanel();
        }
    }

    // Implement the observer interface for node selection events
    onNodeSelected(node) {
        // Properties panel doesn't need to do anything special when nodes are selected
        // The double-click handler takes care of showing the panel
    }

    onNodeDeselected(node) {
        // When a node is deselected, close the properties panel if it's for that node
        if (this.activeNode === node) {
            this.hidePanel();
        }
    }

    setupDOMEventListeners() {
        this.saveButton.addEventListener('click', () => {
            this.saveProperties();
        });

        this.panel.addEventListener('click', (e) => {
            e.stopPropagation();
        });
    }

    showPanel(node, plugin) {
        this.activeNode = node;
        
        // Calculate position for optimal placement
        const nodeRect = node.group.getBoundingClientRect();
        const panelWidth = 220;
        const panelHeight = 400;
        
        let dialogX = nodeRect.right + 10;
        let dialogY = nodeRect.top;
        
        if (dialogX + panelWidth > window.innerWidth) {
            dialogX = nodeRect.left - panelWidth - 10;
        }
        
        if (dialogY + panelHeight > window.innerHeight) {
            dialogY = window.innerHeight - panelHeight - 10;
        }
        
        if (dialogY < 10) {
            dialogY = 10;
        }
        
        this.panel.style.left = dialogX + 'px';
        this.panel.style.top = dialogY + 'px';

        // Hide all groups first
        const allGroups = this.panel.querySelectorAll('.prop-group, #label-group, #compute-group, #lb-group');
        allGroups.forEach(g => g.style.display = 'none');

        const shownGroups = new Set();

        // Set up properties based on plugin definition
        for (const prop of plugin.props) {
            const group = this.panel.querySelector(`[data-group='${prop.group}']`);
            const input = this.panel.querySelector(`[name='${prop.name}']`);

            // Show group once
            if (group && !shownGroups.has(group)) {
                group.style.display = 'block';
                shownGroups.add(group);
            }

            // Set value
            if (input) {
                input.value = node.props[prop.name] ?? prop.default;
            }
        }

        this.saveButton.disabled = false;
        this.panel.style.display = 'block';
        
        setTimeout(() => {
            this.panel.classList.add('visible');
        }, 10);
    }

    hidePanel() {
        if (!this.activeNode) return;

        this.panel.classList.remove('visible');
        
        setTimeout(() => {
            this.panel.style.display = 'none';
        }, 200);
        
        this.activeNode = null;
    }

    saveProperties() {
        if (!this.activeNode) return;

        const node = this.activeNode;
        const panel = this.panel;
        const plugin = PluginRegistry.get(node.type);

        if (!plugin || !plugin.props) {
            this.hidePanel();
            return;
        }

        // Loop through plugin-defined props and update the node
        for (const prop of plugin.props) {
            const input = panel.querySelector(`[name='${prop.name}']`);
            if (!input) continue;

            let value;
            if (prop.type === 'number') {
                value = parseFloat(input.value);
                if (isNaN(value)) value = prop.default ?? 0;
            } else {
                value = input.value;
            }

            node.props[prop.name] = value;
            if (prop.name === 'label') {
                node.updateLabel(value);
            }
        }

        this.hidePanel();
    }
}

/**
 * Selection Manager - Implements node selection observer interface
 */
export class SelectionManager {
    constructor() {
        this.selectedNode = null;
        this.selectedConnection = null;
    }

    // Implement the observer interface for node selection
    onNodeSelected(node) {
        this.clearSelection();
        this.selectedNode = node;
        node.select(); // Visual feedback
    }

    onNodeDeselected(node) {
        if (this.selectedNode === node) {
            node.deselect();
            this.selectedNode = null;
        }
    }

    clearSelection() {
        if (this.selectedNode) {
            this.selectedNode.deselect();
            this.selectedNode = null;
        }

        if (this.selectedConnection) {
            this.selectedConnection.deselect();
            this.selectedConnection = null;
        }
    }
}

/**
 * Initialize the observer system
 */
export function initializeObservers(nodePropsPanel, propsSaveBtn) {
    // Create the specific observers (subjects)
    const nodeSelectionSubject = new NodeSelectionObserver();
    const propertiesPanelSubject = new PropertiesPanelObserver();
    const connectionModeSubject = new ConnectionModeObserver();

    // Create the specific observers (listeners)
    const propertiesPanel = new PropertiesPanelManager(nodePropsPanel, propsSaveBtn);
    
    const selectionManager = new SelectionManager();

    // Wire them together - each observer registers for what it cares about
    nodeSelectionSubject.addObserver(selectionManager);
    nodeSelectionSubject.addObserver(propertiesPanel); // Properties panel cares about selection too

    propertiesPanelSubject.addObserver(propertiesPanel);
    
    connectionModeSubject.addObserver(propertiesPanel); // Panel hides when arrow mode enabled

    // Return the subjects so the main app can notify them
    return {
        nodeSelection: nodeSelectionSubject,
        propertiesPanel: propertiesPanelSubject,
        connectionMode: connectionModeSubject
    };
}

/**
 * How the main CanvasApp would use these observers
 */
export class CanvasAppWithObservers {
    constructor() {
        // Initialize observers
        const observers = initializeObservers();
        this.nodeSelectionSubject = observers.nodeSelection;
        this.propertiesPanelSubject = observers.propertiesPanel;
        this.connectionModeSubject = observers.connectionMode;

        this.selectedNode = null;
        this.arrowMode = false;

        this.setupEventListeners();
    }

    setupEventListeners() {
        // Canvas click - clear selection
        this.canvas.addEventListener('click', (e) => {
            if (e.detail > 1) return; // Ignore double-clicks
            
            if (this.selectedNode) {
                this.nodeSelectionSubject.notifyNodeDeselected(this.selectedNode);
                this.selectedNode = null;
            }
        });

        // Arrow mode toggle
        this.arrowToolBtn.addEventListener('click', () => {
            this.arrowMode = !this.arrowMode;
            this.connectionModeSubject.notifyConnectionModeChanged(this.arrowMode);
        });
    }

    // When a node is double-clicked
    onNodeDoubleClick(node) {
        if (!this.arrowMode) {
            this.propertiesPanelSubject.notifyPropertiesPanelRequested(node);
        }
    }

    // When a node is single-clicked
    onNodeSingleClick(node) {
        if (this.selectedNode !== node) {
            if (this.selectedNode) {
                this.nodeSelectionSubject.notifyNodeDeselected(this.selectedNode);
            }
            this.selectedNode = node;
            this.nodeSelectionSubject.notifyNodeSelected(node);
        }
    }
}

/**
 * Key Benefits of This Approach:
 * 
 * 1. TYPE SAFETY: Each observer has a specific interface
 * 2. SINGLE RESPONSIBILITY: Each observer handles ONE concern  
 * 3. NO MAGIC STRINGS: No event type constants that can be mistyped
 * 4. COMPILE-TIME CHECKING: TypeScript/IDE can validate observer interfaces
 * 5. FOCUSED: PropertiesPanelObserver only knows about properties panels
 * 6. TESTABLE: Each observer can be tested with mock implementations
 * 
 * This approach is more verbose but much safer and clearer about
 * what each component is responsible for.
 */
