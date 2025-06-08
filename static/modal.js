let currentEditingConnection = null;

export function showConnectionModal(connection, onSave) {
    const modal = document.getElementById('connection-modal');
    const labelInput = document.getElementById('connection-label');
    const protocolInput = document.getElementById('connection-protocol');
    const saveBtn = document.getElementById('modal-save');
    const cancelBtn = document.getElementById('modal-cancel');
    const closeBtn = document.getElementById('modal-close');

    currentEditingConnection = connection;

    labelInput.value = connection.label || '';
    protocolInput.value = connection.protocol || '';

    modal.style.display = 'flex';
    labelInput.focus();

    const close = () => {
        modal.style.display = 'none';
        currentEditingConnection = null;
    };

    saveBtn.onclick = () => {
        connection.label = labelInput.value.trim();
        connection.protocol = protocolInput.value.trim();
        connection.text.textContent = connection.label;
        connection.protocolText.textContent = connection.protocol;
        connection.updatePosition();
        if (onSave) onSave(connection);
        close();
    };

    cancelBtn.onclick = close;
    closeBtn.onclick = close;
}

