/**
 * Planillas RGM - JavaScript Global (Pico CSS v2 + HTMX Helpers)
 */

document.addEventListener('DOMContentLoaded', () => {
    initHTMXConfirmModal();
});

/**
 * Intercepta el evento HTMX `htmx:confirm` para elementos con data-confirm-title o data-confirm-message
 */
function initHTMXConfirmModal() {
    document.body.addEventListener('htmx:confirm', (evt) => {
        const triggerEl = evt.detail.elt;
        const confirmTitle = triggerEl.getAttribute('data-confirm-title');
        const confirmMsg = triggerEl.getAttribute('data-confirm-message');

        // Si no tiene atributos de confirmación custom, dejar flujo estándar HTMX
        if (!confirmTitle && !confirmMsg) {
            return;
        }

        // Cancelar el diálogo nativo confirm() de HTMX
        evt.preventDefault();

        const modal = document.getElementById('modal-confirmacion-global');
        if (!modal) {
            if (window.confirm(confirmMsg || confirmTitle)) {
                evt.detail.issueRequest();
            }
            return;
        }

        const titleEl = document.getElementById('confirm-modal-title');
        const msgEl = document.getElementById('confirm-modal-message');
        const badgeEl = document.getElementById('confirm-modal-badge');
        const btnTextoEl = document.getElementById('confirm-modal-btn-texto');
        const btnCancelar = document.getElementById('confirm-modal-btn-cancelar');
        const btnAceptar = document.getElementById('confirm-modal-btn-aceptar');

        const badgeTexto = triggerEl.getAttribute('data-confirm-badge') || 'Advertencia';
        const btnTexto = triggerEl.getAttribute('data-confirm-btn') || 'Sí, Confirmar';

        if (titleEl && titleEl.querySelector('span')) {
            titleEl.querySelector('span').textContent = confirmTitle || 'Confirmar Acción';
        }
        if (msgEl) {
            msgEl.textContent = confirmMsg || '¿Estás seguro de realizar esta acción?';
        }
        if (badgeEl) {
            badgeEl.textContent = badgeTexto;
        }
        if (btnTextoEl) {
            btnTextoEl.textContent = btnTexto;
        }

        const cleanup = () => {
            btnCancelar.removeEventListener('click', onCancel);
            btnAceptar.removeEventListener('click', onConfirm);
            modal.close();
        };

        const onCancel = () => {
            cleanup();
        };

        const onConfirm = () => {
            cleanup();
            evt.detail.issueRequest();
        };

        btnCancelar.addEventListener('click', onCancel);
        btnAceptar.addEventListener('click', onConfirm);

        modal.showModal();
    });
}
