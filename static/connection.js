import { createSVGElement } from './utils.js';

export class Connection {
    static _activeConnection = null;
    static modalSetupDone = false;

    constructor(startNode, endNode, label, protocol, app, tls, capacity = 1000) {
        this.start = startNode;
        this.end = endNode;
        this.app = app;
        this.label = label;
        this.protocol = protocol;
        this.direction = "forward";
        this.tls = tls;
        this.capacity = capacity;
        this.line = createSVGElement('line', {
            stroke: '#ccc', 'stroke-width': 2, 'marker-end': 'url(#arrowhead-end)'
        });
        this.hitbox = createSVGElement('circle', {
            r: 12,
            fill: 'transparent',
            cursor: 'pointer',
        });
        this.text = createSVGElement('text', {
            'text-anchor': 'middle', 'font-size': 12, fill: '#ccc'
        });
        this.text.textContent = label;
        this.protocolText = createSVGElement('text', {
            'text-anchor': 'middle',
            'font-size': 10,
            fill: '#888'
        });
        this.protocolText.textContent = this.protocol || '';
        app.canvas.appendChild(this.line);
        app.canvas.appendChild(this.text);
        app.canvas.appendChild(this.protocolText)
        app.canvas.appendChild(this.hitbox);

        this.updatePosition();

        this.selected = false;
        this.line.addEventListener('click', (e) => {
            e.stopPropagation();
            // Clear node selection via observer
            if (this.app.selectedNode) {
                this.app.nodeSelectionSubject.notifyNodeDeselected(this.app.selectedNode);
                this.app.selectedNode = null;
            }
            // Clear any previously selected connection
            if (this.app.selectedConnection) {
                this.app.selectedConnection.deselect();
            }
            this.select();
        });

        this.line.addEventListener('dblclick', (e) => {
            e.stopPropagation();
            this.toggleDirection();
        });

        this.text.addEventListener('dblclick', (e) => {
            e.stopPropagation();
            this.openEditModal()
        });

        this.protocolText.addEventListener('click', (e) => {
            e.stopPropagation();
            this.openEditModal();
        });
    }

    updatePosition() {
        const s = this.start.getConnectionPointToward(this.end);
        const e = this.end.getConnectionPointToward(this.start);

        this.line.setAttribute('x1', s.x);
        this.line.setAttribute('y1', s.y);
        this.line.setAttribute('x2', e.x);
        this.line.setAttribute('y2', e.y);

        // update text position (midpoint of the line)
        const midX = (s.x + e.x) / 2;
        const midY = (s.y + e.y) / 2;
        this.text.setAttribute('x', midX);
        this.text.setAttribute('y', midY - 6);

        // update arrowheads based on direction
        if (this.direction === 'forward') {
            this.line.setAttribute('marker-start', '');
            this.line.setAttribute('marker-end', 'url(#arrowhead-end)');
        } else if (this.direction === 'backward') {
            this.line.setAttribute('marker-start', 'url(#arrowhead-start)');
            this.line.setAttribute('marker-end', '');
        } else if (this.direction === 'bidirectional') {
            this.line.setAttribute('marker-start', 'url(#arrowhead-start)');
            this.line.setAttribute('marker-end', 'url(#arrowhead-end)');
        }

        // Hitbox position (depends on direction)
        let hbX, hbY;
        if (this.direction === 'forward') {
            hbX = e.x;
            hbY = e.y;
        } else if (this.direction === 'backward') {
            hbX = s.x;
            hbY = s.y;
        } else {
            hbX = midX;
            hbY = midY;
        }

        this.hitbox.setAttribute('cx', hbX);
        this.hitbox.setAttribute('cy', hbY);

        this.protocolText.setAttribute('x', midX);
        this.protocolText.setAttribute('y', midY + 12);
    }

    toggleDirection() {
        const order = ['forward', 'backward', 'bidirectional'];
        const currentIndex = order.indexOf(this.direction);
        const nextIndex = (currentIndex + 1) % order.length;
        this.direction = order[nextIndex]
        this.updatePosition();
    }

    select() {
        // Clear node selection via observer
        if (this.app.selectedNode) {
            this.app.nodeSelectionSubject.notifyNodeDeselected(this.app.selectedNode);
            this.app.selectedNode = null;
        }
        // Clear any previously selected connection
        if (this.app.selectedConnection) {
            this.app.selectedConnection.deselect();
        }
        this.selected = true;
        this.line.setAttribute('stroke', '#007bff');
        this.line.setAttribute('stroke-width', 3);
        this.app.selectedConnection = this;
    }

    deselect() {
        this.selected = false;
        this.line.setAttribute('stroke', '#333');
        this.line.setAttribute('stroke-width', 2);
    }

    static setupModal(app) {
        if (Connection.modalSetupDone) return;
        Connection.modalSetupDone = true;

        Connection.modal = document.getElementById('connection-modal');
        Connection.labelInput = document.getElementById('connection-label');
        Connection.tlsCheckbox = document.getElementById('connection-tls');
        Connection.protocolInput = document.getElementById('connection-protocol');
        Connection.capacityInput = document.getElementById('connection-capacity');
        Connection.saveBtn = document.getElementById('connection-save');
        Connection.cancelBtn = document.getElementById('connection-cancel');

        Connection.saveBtn.addEventListener('click', () => {
            const label = Connection.labelInput.value.trim();
            const protocol = Connection.protocolInput.value.trim();
            const tls = Connection.tlsCheckbox.checked;
            const capacity = parseInt(Connection.capacityInput.value.trim(), 10) || 1000;
            if (!label || !protocol) return;

            if (Connection._activeConnection) {
                // Editing an existing connection
                const conn = Connection._activeConnection;
                conn.label = label;
                conn.protocol = protocol;
                conn.text.textContent = label;
                conn.tls = tls;
                conn.capacity = capacity;
                conn.protocolText.textContent = protocol;
                conn.updatePosition();
            } else if (app.pendingConnection) {
                // Creating a new connection
                const { start, end } = app.pendingConnection;
                const conn = new Connection(start, end, label, protocol, app, tls, capacity);
                app.connections.push(conn);
            }

            Connection.modal.style.display = 'none';
            app.pendingConnection = null;
            Connection._activeConnection = null;

            if (app.connectionStart) {
                app.connectionStart.group.classList.remove('selected');
                app.connectionStart = null;
            }
        });

        Connection.cancelBtn.addEventListener('click', () => {
            Connection.modal.style.display = 'none';
            app.pendingConnection = null;

            if (app.connectionStart) {
                app.connectionStart.group.classList.remove('selected');
                app.connectionStart = null;
            }
        });
    }

    static handleClick(nodeObj, app) {
        Connection.setupModal(app);

        if (!app.connectionStart) {
            app.connectionStart = nodeObj;
            nodeObj.group.classList.add('selected');
        } else if (app.connectionStart === nodeObj) {
            app.connectionStart.group.classList.remove('selected');
            app.connectionStart = null;
        } else {
            app.pendingConnection = { start: app.connectionStart, end: nodeObj };
            Connection.labelInput.value = 'Read traffic';
            Connection.protocolInput.value = 'HTTP';
            Connection.tlsCheckbox.checked = false;
            Connection.capacityInput.value = '1000';
            Connection.modal.style.display = 'block';
        }
    }

    openEditModal() {
        Connection.setupModal(this.app);
        Connection._activeConnection = this;

        Connection.labelInput.value = this.label;
        Connection.protocolInput.value = this.protocol;
        Connection.tlsCheckbox.checked = this.tls;
        Connection.capacityInput.value = this.capacity || 1000;
        Connection.modal.style.display = 'block';
    }

}
