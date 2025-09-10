/**
 * Command Pattern Implementation
 * 
 * This system encapsulates user actions as command objects, making the codebase
 * more maintainable and providing a foundation for features like undo/redo.
 */

import { PluginRegistry } from './pluginRegistry.js';
import { generateDefaultProps } from './utils.js';
import { ComponentNode } from './node.js';

// Base Command interface
export class Command {
    /**
     * Execute the command
     * @param {CanvasApp} app - The application context
     */
    execute(app) {
        throw new Error('Command must implement execute() method');
    }

    /**
     * Optional: Undo the command (for future undo/redo system)
     * @param {CanvasApp} app - The application context
     */
    undo(app) {
        // Optional implementation - most commands won't need this initially
    }

    /**
     * Optional: Get command description for logging/debugging
     */
    getDescription() {
        return this.constructor.name;
    }
}

// Command Invoker - manages command execution and history
export class CommandInvoker {
    constructor(app) {
        this.app = app;
        this.history = [];
        this.maxHistorySize = 100; // Prevent memory leaks
    }

    /**
     * Execute a command and add it to history
     * @param {Command} command 
     */
    execute(command) {
        try {
            command.execute(this.app);

            // Add to history (for future undo system)
            this.history.push(command);
            if (this.history.length > this.maxHistorySize) {
                this.history.shift();
            }
        } catch (error) {
            throw error;
        }
    }

    /**
     * Future: Undo last command
     */
    undo() {
        if (this.history.length === 0) return;

        const command = this.history.pop();
        if (command.undo) {
            command.undo(this.app);
        }
    }

    /**
     * Get command history for debugging
     */
    getHistory() {
        return this.history.map(cmd => cmd.getDescription());
    }
}

// =============================================================================
// TAB NAVIGATION COMMANDS
// =============================================================================

export class SwitchToResourcesTabCommand extends Command {
    execute(app) {
        const requirementstab = app.tabs[1];
        const resourcestab = app.tabs[2];

        requirementstab.checked = false;
        resourcestab.checked = true;
    }
}

export class SwitchToDesignTabCommand extends Command {
    execute(app) {
        const requirementstab = app.tabs[1];
        const designtab = app.tabs[1]; // Note: This looks like a bug in original - should be tabs[0]?

        requirementstab.checked = false;
        designtab.checked = true;
    }
}

// =============================================================================
// TOOL COMMANDS
// =============================================================================

export class ToggleArrowModeCommand extends Command {
    execute(app) {
        app.arrowMode = !app.arrowMode;

        if (app.arrowMode) {
            app.arrowToolBtn.classList.add('active');
            // Use observer to notify that arrow mode is enabled (will hide props panel)
            app.connectionModeSubject.notifyConnectionModeChanged(true);
        } else {
            app.arrowToolBtn.classList.remove('active');
            if (app.connectionStart) {
                app.connectionStart.group.classList.remove('selected');
                app.connectionStart = null;
            }
            // Use observer to notify that arrow mode is disabled
            app.connectionModeSubject.notifyConnectionModeChanged(false);
        }
    }
}

// =============================================================================
// CHAT COMMANDS
// =============================================================================

export class StartChatCommand extends Command {
    execute(app) {
        const scheme = location.protocol === "https:" ? "wss://" : "ws://";

        app.ws = new WebSocket(scheme + location.host + "/ws");

        app.ws.onopen = () => {
            app.ws.send(JSON.stringify({
                'designPayload': JSON.stringify(app.exportDesign()),
                'message': ''
            }));
        };

        app.ws.onmessage = (e) => {
            app.chatLoadingIndicator.style.display = 'none';
            app.chatTextField.disabled = false;
            app.chatTextField.focus();
            const message = document.createElement('p');
            message.innerHTML = e.data;
            message.className = "other";
            app.chatMessages.insertBefore(message, app.chatLoadingIndicator);
        };

        app.ws.onerror = (err) => {
            console.log("ws error:", err);
            app._scheduleReconnect();
        };

        app.ws.onclose = () => {
            console.log("leaving chat...");
            app.ws = null;
            app._scheduleReconnect();
        };
    }
}

export class SendChatMessageCommand extends Command {
    constructor(message) {
        super();
        this.message = message;
    }

    execute(app) {
        const messageElement = document.createElement('p');
        messageElement.innerHTML = this.message;
        messageElement.className = "me";
        app.chatMessages.insertBefore(messageElement, app.chatLoadingIndicator);

        app.ws.send(JSON.stringify({
            'message': this.message,
            'designPayload': JSON.stringify(app.exportDesign()),
        }));

        app.chatTextField.value = '';
        app.chatLoadingIndicator.style.display = 'block';
    }
}

// =============================================================================
// DRAG & DROP COMMANDS
// =============================================================================

export class HandleDragStartCommand extends Command {
    constructor(event) {
        super();
        this.event = event;
    }

    execute(app) {
        const type = this.event.target.getAttribute('data-type');
        const plugin = PluginRegistry.get(type);

        if (!plugin) return;

        this.event.dataTransfer.setData('text/plain', type);
    }
}

export class HandleDragEndCommand extends Command {
    constructor(event) {
        super();
        this.event = event;
    }

    execute(app) {
        if (this.event.target.classList.contains('component-icon')) {
            this.event.target.classList.remove('dragging');
        }
    }
}

export class DropComponentCommand extends Command {
    constructor(event) {
        super();
        this.event = event;
    }

    execute(app) {
        const type = this.event.dataTransfer.getData('text/plain');
        const plugin = PluginRegistry.get(type);
        if (!plugin) return;

        const pt = app.canvas.createSVGPoint();
        pt.x = this.event.clientX;
        pt.y = this.event.clientY;

        const svgP = pt.matrixTransform(app.canvas.getScreenCTM().inverse());
        const x = svgP.x - app.componentSize.width / 2;
        const y = svgP.y - app.componentSize.height / 2;

        const props = generateDefaultProps(plugin);
        const node = new ComponentNode(type, x, y, app, props);
        node.x = x;
        node.y = y;
    }
}

// =============================================================================
// SIMULATION COMMANDS
// =============================================================================

export class RunSimulationCommand extends Command {
    async execute(app) {
        const designData = app.exportDesign();

        // Try to get level info from URL or page context
        const levelInfo = app.getLevelInfo();

        const requestBody = {
            design: designData,
            ...levelInfo
        };

        console.log('Sending design to simulation:', JSON.stringify(requestBody));

        // Disable button and show loading state
        app.runButton.disabled = true;
        app.runButton.textContent = 'Running Simulation...';

        try {
            const response = await fetch('/simulate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            // Check if response is a redirect (status 303)
            if (response.redirected || response.status === 303) {
                // Follow the redirect to the result page
                window.location.href = response.url;
                return;
            }

            // Fallback: try to parse as JSON for backward compatibility
            const result = await response.json();
            console.log('result', result);
            if (result.passed && result.success) {
                console.log('Simulation successful:', result);
                app.showResults(result);
            } else {
                console.error('Simulation failed:', result.Error);
                app.showError(result.Error || 'Simulation failed');
            }

        } catch (error) {
            console.error('Network error:', error);
            app.showError('Failed to run simulation: ' + error.message);
        } finally {
            // Re-enable button
            app.runButton.disabled = false;
            app.runButton.textContent = 'Test Design';
        }
    }
}

// =============================================================================
// CANVAS INTERACTION COMMANDS
// =============================================================================

export class HandleCanvasClickCommand extends Command {
    constructor(event) {
        super();
        this.event = event;
    }

    execute(app) {
        // Delegate to current state
        app.stateMachine.handleCanvasClick(this.event);
    }
}

export class SaveNodePropertiesCommand extends Command {
    execute(app) {
        if (!app.activeNode) return;

        const node = app.activeNode;
        const panel = app.nodePropsPanel;
        const plugin = PluginRegistry.get(node.type);

        if (!plugin || !plugin.props) {
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
    }
}

export class DeleteSelectionCommand extends Command {
    constructor(key) {
        super();
        this.key = key;
    }

    execute(app) {
        if (this.key === 'Backspace' || this.key === 'Delete') {
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
}
