/**
 * Planillas RGM - JavaScript Global (Pico CSS v2 + HTMX Helpers)
 */

document.addEventListener('DOMContentLoaded', () => {
    initHTMXConfirmModal();
    initTomSelectAuto();
    initSwappedScriptsHandler();
});

/**
 * Inicialización centralizada y automática de componentes TomSelect
 * Busca elementos .select-con-buscador excluyendo selectores personalizados (data-custom-tomselect="true" o #select-agregar-concepto)
 */
function initTomSelectAuto() {
    const initSelectsInContainer = (container) => {
        if (!container || typeof TomSelect === 'undefined') return;
        const selects = container.querySelectorAll('.select-con-buscador:not([data-custom-tomselect]):not(#select-agregar-concepto)');
        selects.forEach((el) => {
            if (!el.tomselect) {
                const isMulti = el.hasAttribute('multiple');
                new TomSelect(el, {
                    create: false,
                    allowEmptyOption: true,
                    plugins: isMulti ? ['remove_button'] : ['dropdown_input'],
                    maxItems: isMulti ? null : 1
                });
            }
        });
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => initSelectsInContainer(document.body));
    } else {
        initSelectsInContainer(document.body);
    }

    document.body.addEventListener('htmx:afterSettle', (evt) => {
        const target = evt.detail.target || document.body;
        initSelectsInContainer(target);
    });
}

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

/**
 * Ejecuta scripts que vengan dentro de fragmentos inyectados por HTMX
 * y dispara la inicialización de vistas dinámicas.
 */
function initSwappedScriptsHandler() {
    const handleSwap = (container) => {
        if (!container || !container.querySelectorAll) return;

        const scripts = container.querySelectorAll('script');
        scripts.forEach((oldScript) => {
            if (oldScript.dataset.executed) return;
            oldScript.dataset.executed = 'true';

            const type = oldScript.getAttribute('type');
            if (!type || type === 'text/javascript' || type === 'application/javascript') {
                const newScript = document.createElement('script');
                Array.from(oldScript.attributes).forEach((attr) => newScript.setAttribute(attr.name, attr.value));
                newScript.textContent = oldScript.textContent;
                oldScript.parentNode.replaceChild(newScript, oldScript);
            }
        });

        // Inicializar totales de tabla SUNAT si está presente
        if (typeof window.actualizarTotalesConceptosSunat === 'function' && container.querySelector && container.querySelector('#tabla-conceptos-sunat')) {
            window.actualizarTotalesConceptosSunat();
        }
    };

    document.body.addEventListener('htmx:afterSettle', (evt) => {
        const target = evt.detail.target || document.body;
        handleSwap(target);
    });
}

/* =========================================================================
 * Módulo de Auditoría y Declaración PDT PLAME (SUNAT)
 * ========================================================================= */

window.tipoConceptoSunatActivo = '';

window.actualizarTotalesConceptosSunat = function () {
    let devengado = 0;
    let pagado = 0;
    let visibles = 0;
    const rows = document.querySelectorAll('#tabla-conceptos-sunat tbody tr.fila-concepto-item');
    if (!rows || rows.length === 0) return;

    rows.forEach(row => {
        if (!row.hidden && row.style.display !== 'none') {
            visibles++;
            devengado += parseFloat(row.getAttribute('data-devengado') || 0);
            pagado += parseFloat(row.getAttribute('data-pagado') || 0);
        }
    });

    const elDev = document.getElementById('total-devengado-pie');
    const elPag = document.getElementById('total-pagado-pie');
    const elContador = document.getElementById('contador-conceptos-visibles');

    if (elDev) elDev.textContent = 'S/ ' + devengado.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    if (elPag) elPag.textContent = 'S/ ' + pagado.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    if (elContador) elContador.textContent = visibles;
};

window.filtrarTipoConcepto = function (tipo, btnEl) {
    window.tipoConceptoSunatActivo = (tipo || '').toUpperCase().trim();

    const btnContainer = document.getElementById('grupo-botones-filtro-tipo');
    if (btnContainer) {
        btnContainer.querySelectorAll('.btn-filtro-tipo').forEach(b => {
            b.classList.add('outline', 'secondary');
        });
        if (btnEl) {
            btnEl.classList.remove('outline', 'secondary');
        }
    }
    window.filtrarTablaConceptosSunat();
};

window.filtrarTablaConceptosSunat = function () {
    const input = document.getElementById('filtro-concepto-sunat');
    const filter = input ? input.value.toLowerCase().trim() : '';
    const tipoFiltro = (window.tipoConceptoSunatActivo || '').toUpperCase().trim();

    const rows = document.querySelectorAll('#tabla-conceptos-sunat tbody tr.fila-concepto-item');
    let visibles = 0;

    rows.forEach(row => {
        const rowTipo = (row.getAttribute('data-tipo') || '').toUpperCase().trim();
        const rowNombre = (row.getAttribute('data-nombre') || '').toLowerCase();
        const rowCodigo = (row.getAttribute('data-codigo') || '').toLowerCase();
        const rowDesc = (row.getAttribute('data-desc') || '').toLowerCase();
        const rowBadges = Array.from(row.querySelectorAll('.badge, mark')).map(b => b.textContent.toLowerCase()).join(' ');

        const rowSearchable = (rowNombre + ' ' + rowCodigo + ' ' + rowDesc + ' ' + rowTipo.toLowerCase() + ' ' + rowBadges).toLowerCase();

        const matchTipo = (tipoFiltro === '' || rowTipo === tipoFiltro || (tipoFiltro === 'RETENCION' && (rowTipo === 'DESCUENTO' || rowTipo === 'RETENCION')));
        const matchTexto = (filter === '' || rowSearchable.includes(filter));

        if (matchTipo && matchTexto) {
            row.style.display = '';
            row.hidden = false;
            visibles++;
        } else {
            row.style.display = 'none';
            row.hidden = true;
        }
    });

    const noResultsRow = document.getElementById('fila-sin-resultados-filtro');
    if (noResultsRow) {
        const mostrarSinResultados = (visibles === 0 && rows.length > 0);
        noResultsRow.style.display = mostrarSinResultados ? '' : 'none';
        noResultsRow.hidden = !mostrarSinResultados;
    }

    window.actualizarTotalesConceptosSunat();
};

window.filtrarTrabajadoresPlame = function (termino) {
    const query = (termino || '').toLowerCase().trim();
    const filas = document.querySelectorAll('.fila-trabajador-item');
    let visibles = 0;

    filas.forEach(function (fila) {
        const dni = (fila.getAttribute('data-dni') || '').toLowerCase();
        const nombre = (fila.getAttribute('data-nombre') || '').toLowerCase();

        if (!query || dni.includes(query) || nombre.includes(query)) {
            fila.style.display = '';
            fila.hidden = false;
            visibles++;
        } else {
            fila.style.display = 'none';
            fila.hidden = true;
        }
    });

    const cont = document.getElementById('contador-trabajadores-visibles');
    if (cont) cont.textContent = visibles;

    const filaSinResultados = document.getElementById('fila-sin-trabajadores-busqueda');
    if (filaSinResultados) {
        filaSinResultados.style.display = (visibles === 0 && filas.length > 0) ? '' : 'none';
    }
};

