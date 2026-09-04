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

        // Inicializar totales de tabla de colaboradores SUNAT si está presente
        if (typeof window.actualizarTotalesTrabajadoresPlame === 'function' && container.querySelector && container.querySelector('#tabla-trabajadores-plame')) {
            window.actualizarTotalesTrabajadoresPlame();
        }

        // Inicializar totales de tabla de edición de conceptos de trabajador si está presente
        if (typeof window.actualizarTotalesEdicionTrabajador === 'function' && container.querySelector && container.querySelector('#tabla-edicion-conceptos-trabajador')) {
            window.actualizarTotalesEdicionTrabajador();
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

window.actualizarTotalesTrabajadoresPlame = function () {
    let devengado = 0;
    let pagado = 0;
    const rows = document.querySelectorAll('#tabla-trabajadores-plame tbody tr.fila-trabajador-item');
    if (!rows || rows.length === 0) return;

    rows.forEach(row => {
        if (!row.hidden && row.style.display !== 'none') {
            devengado += parseFloat(row.getAttribute('data-devengado') || 0);
            pagado += parseFloat(row.getAttribute('data-pagado') || 0);
        }
    });

    const elDev = document.getElementById('total-devengado-trabajadores');
    const elPag = document.getElementById('total-pagado-trabajadores');

    if (elDev) elDev.textContent = 'S/ ' + devengado.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    if (elPag) elPag.textContent = 'S/ ' + pagado.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
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

    window.actualizarTotalesTrabajadoresPlame();
};

window.agregarFilaDesdoblamientoTrabajador = function () {
    const tbody = document.getElementById('tbody-plame-conceptos-filas');
    if (!tbody) return;

    const primeraFila = tbody.querySelector('tr.fila-concepto-trabajador-item');
    if (!primeraFila) return;

    const nuevaFila = primeraFila.cloneNode(true);

    // Actualizar etiqueta del concepto origen para indicar desdoblamiento
    const celdaOrigen = nuevaFila.querySelector('td:first-child');
    if (celdaOrigen) {
        celdaOrigen.innerHTML = '<mark class="badge badge-warning font-xxs">Sub-desdoblamiento</mark>';
    }

    // Resetear código SUNAT
    const valInput = nuevaFila.querySelector('.input-codigo-sunat-val');
    const badge = nuevaFila.querySelector('.badge-codigo-sunat-display');
    const desc = nuevaFila.querySelector('.text-codigo-sunat-desc');
    if (valInput) valInput.value = '';
    if (badge) badge.textContent = '----';
    if (desc) {
        desc.textContent = '-- Seleccionar Código --';
        desc.title = '';
    }

    // Resetear montos a 0.00
    const devInput = nuevaFila.querySelector('input[name="monto_devengado[]"]');
    const pagInput = nuevaFila.querySelector('input[name="monto_pagado[]"]');
    const vacHidden = nuevaFila.querySelector('.input-es-vacacional-val');
    const vacCheck = nuevaFila.querySelector('input[type="checkbox"]');
    if (devInput) devInput.value = '0.00';
    if (pagInput) pagInput.value = '0.00';
    if (vacHidden) vacHidden.value = 'false';
    if (vacCheck) vacCheck.checked = false;

    tbody.appendChild(nuevaFila);
    window.actualizarTotalesEdicionTrabajador();

    // Abrir automáticamente el modal buscador para la nueva fila
    const btnCambiar = nuevaFila.querySelector('.btn-cambiar-sunat');
    if (btnCambiar) {
        window.abrirBuscadorCodigoSunat(btnCambiar);
    }
};

window.eliminarFilaConceptoTrabajador = function (btn) {
    const tr = btn.closest('tr');
    if (!tr) return;
    const tbody = tr.closest('tbody');
    if (tbody && tbody.querySelectorAll('tr').length <= 1) {
        alert('Debe mantener al menos una línea de concepto tributario para el colaborador.');
        return;
    }
    tr.remove();
    window.actualizarTotalesEdicionTrabajador();
};

window.actualizarTotalesEdicionTrabajador = function () {
    let devengado = 0;
    let pagado = 0;
    const filas = document.querySelectorAll('#tabla-edicion-conceptos-trabajador tbody tr.fila-concepto-trabajador-item');

    filas.forEach(fila => {
        const dev = parseFloat(fila.querySelector('input[name="monto_devengado[]"]')?.value || 0);
        const pag = parseFloat(fila.querySelector('input[name="monto_pagado[]"]')?.value || 0);
        devengado += isNaN(dev) ? 0 : dev;
        pagado += isNaN(pag) ? 0 : pag;
    });

    const elDev = document.getElementById('total-edicion-devengado');
    const elPag = document.getElementById('total-edicion-pagado');
    const contadorLineas = document.getElementById('contador-lineas-edicion');

    if (elDev) elDev.textContent = 'S/ ' + devengado.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    if (elPag) elPag.textContent = 'S/ ' + pagado.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    if (contadorLineas) contadorLineas.textContent = filas.length;
};

// ==============================================================================
// Componente Reutilizable: Modal Buscador de Códigos SUNAT (Tabla 22 - PLAME)
// ==============================================================================
window.filaEdicionActiva = null;
window.tipoFiltroSunatModalActivo = '';

window.abrirBuscadorCodigoSunat = function (btn) {
    const tr = btn ? btn.closest('tr') : null;
    if (!tr) return;
    window.filaEdicionActiva = tr;

    const modal = document.getElementById('modal-buscador-codigo-sunat');
    if (!modal) return;

    // Resetear filtro de búsqueda
    const input = document.getElementById('filtro-buscador-sunat-modal');
    if (input) {
        input.value = '';
    }

    // Resetear pestañas de filtro a "Todos"
    window.tipoFiltroSunatModalActivo = '';
    const tabContainer = document.getElementById('tabs-filtro-sunat-modal');
    if (tabContainer) {
        tabContainer.querySelectorAll('.btn-tab-sunat').forEach(b => {
            if (b.getAttribute('data-filtro') === '') {
                b.className = 'btn-compact font-xs btn-tab-sunat';
            } else {
                b.className = 'outline secondary btn-compact font-xs btn-tab-sunat';
            }
        });
    }

    window.filtrarCatalogoSunatModal('');
    modal.showModal();

    setTimeout(() => {
        if (input) input.focus();
    }, 50);
};

window.filtrarCatalogoSunatModal = function (termino) {
    const query = (termino || '').toLowerCase().trim();
    const prefijo = window.tipoFiltroSunatModalActivo || '';
    const filas = document.querySelectorAll('#tabla-catalogo-sunat-modal tbody tr.fila-catalogo-sunat-item');
    let visibles = 0;

    filas.forEach(fila => {
        const cod = (fila.getAttribute('data-codigo') || '').toLowerCase();
        const desc = (fila.getAttribute('data-desc') || '').toLowerCase();

        // Filtro por prefijo de pestaña
        let cumplePrefijo = true;
        if (prefijo === '01') {
            const codNum = parseInt(cod, 10);
            cumplePrefijo = (codNum >= 100 && codNum < 600);
        } else if (prefijo === '06') {
            const codNum = parseInt(cod, 10);
            cumplePrefijo = (codNum >= 600 && codNum < 800);
        } else if (prefijo === '08') {
            const codNum = parseInt(cod, 10);
            cumplePrefijo = (codNum >= 800 && codNum < 1000);
        }

        // Filtro por texto
        const cumpleTexto = !query || cod.includes(query) || desc.includes(query);

        if (cumplePrefijo && cumpleTexto) {
            fila.style.display = '';
            visibles++;
        } else {
            fila.style.display = 'none';
        }
    });

    const contador = document.getElementById('contador-maestros-sunat-visibles');
    if (contador) contador.textContent = visibles;

    const filaVacia = document.getElementById('fila-sin-catalogo-sunat');
    if (filaVacia) {
        filaVacia.style.display = (visibles === 0 && filas.length > 0) ? '' : 'none';
    }
};

window.filtrarTipoCatalogoSunatModal = function (filtro, btnEl) {
    window.tipoFiltroSunatModalActivo = filtro || '';

    const tabContainer = document.getElementById('tabs-filtro-sunat-modal');
    if (tabContainer && btnEl) {
        tabContainer.querySelectorAll('.btn-tab-sunat').forEach(b => {
            b.className = 'outline secondary btn-compact font-xs btn-tab-sunat';
        });
        btnEl.className = 'btn-compact font-xs btn-tab-sunat';
    }

    const input = document.getElementById('filtro-buscador-sunat-modal');
    window.filtrarCatalogoSunatModal(input ? input.value : '');
};

window.seleccionarCodigoSunat = function (codigo, descripcion, tipo) {
    if (!window.filaEdicionActiva) return;

    const tr = window.filaEdicionActiva;

    // Actualizar input oculto
    const valInput = tr.querySelector('.input-codigo-sunat-val');
    if (valInput) valInput.value = codigo;

    // Actualizar badge visual
    const badge = tr.querySelector('.badge-codigo-sunat-display');
    if (badge) badge.textContent = codigo;

    // Actualizar descripción
    const desc = tr.querySelector('.text-codigo-sunat-desc');
    if (desc) {
        desc.textContent = descripcion;
        desc.title = descripcion;
    }

    // Sincronizar select de naturaleza si coincide
    if (tipo) {
        const selectTipo = tr.querySelector('select[name="tipo_concepto[]"]');
        if (selectTipo) {
            const upperTipo = tipo.toUpperCase();
            for (let opt of selectTipo.options) {
                if (opt.value === upperTipo) {
                    selectTipo.value = upperTipo;
                    break;
                }
            }
        }
    }

    const modal = document.getElementById('modal-buscador-codigo-sunat');
    if (modal) modal.close();
};



