/**
 * Generates a unique node ID like "node-1", "node-2", etc.
 * @returns {string}
 */
let idCounter = 0;
export function generateNodeId() {
    return `node-${idCounter++}`;
}

/**
 * Creates an SVG element with the given tag and attributes.
 * @param {string} tag - The SVG tag name (e.g., 'rect', 'line').
 * @param {Object} attrs - An object of attribute key-value pairs.
 * @returns {SVGElement}
 */
export function createSVGElement(tag, attrs) {
    const elem = document.createElementNS('http://www.w3.org/2000/svg', tag);
    for (let key in attrs) {
        elem.setAttribute(key, attrs[key]);
    }
    return elem;
}

export function generateDefaultProps(plugin) {
    const defaults = {};
    plugin.props?.forEach(p => {
        defaults[p.name] = p.default;
    });
    return defaults;
}
