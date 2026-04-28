# Plan de trabajo 2 
## 1. Importación de Inasistencias y Tardanzas (Excel)
Digitar faltas y minutos de tardanza trabajador por trabajador es inviable en entidades públicas grandes. Usaremos Excel para automatizarlo.

Librería a usar: Integraremos github.com/xuri/excelize/v2, que es el estándar de oro en Go para leer y escribir archivos .xlsx de forma nativa y ultra rápida.

Paso 1 (Plantilla Estándar): Definiremos una estructura simple para el Excel (Ej. Columna A: DNI, Columna B: Días de Falta, Columna C: Minutos de Tardanza).

Paso 2 (La UI): En la vista de asistencias, agregaremos un formulario con enctype="multipart/form-data" para subir el archivo. Aquí no usaremos HTMX para el envío, sino un formulario tradicional, ya que la carga de archivos binarios es más estable así.

Paso 3 (El Handler y Servicio): El backend recibirá el Excel, lo leerá fila por fila, buscará el trabajador_id correspondiente a cada DNI y hará un guardado masivo en la tabla asistencias para el periodo seleccionado.

## 2. Edición del Catálogo de Conceptos Locales (conceptos_tenant)
Las municipalidades cambian constantemente los clasificadores presupuestales o deciden renombrar conceptos (Ej. pasar de "Remuneración CAS" a "Contrato CAS - Sueldo Base").

Paso 1 (Repositorio): Crearemos la función ActualizarConcepto que permitirá modificar el nombre_personalizado, el clasificador_id y el estado activo.

Paso 2 (La UI con HTMX): Replicaremos la "magia" que usamos en los montos de la plaza. Pondremos un botón "✏️ Editar" al lado de cada concepto en la tabla de configuración. Al hacer clic, la fila entera se convertirá en un mini-formulario (Inline Editing) con menús desplegables para cambiar el clasificador al vuelo.

Paso 3 (Protección): Nos aseguraremos de que no se pueda modificar el Código Maestro (SUNAT), ya que eso rompería la lógica de nuestro motor de cálculo.

## 3. Ciclo de Vida de la Planilla (Cambio de Estado)
Actualmente, las planillas viven en un eterno BORRADOR. Necesitamos poder "Cerrarlas" para que queden congeladas y listas para el pago.

Paso 1 (Estados definidos): Manejaremos dos estados principales: BORRADOR (permite recalcular indefinidamente) y CERRADA (bloquea cualquier modificación).

Paso 2 (Repositorio y Reglas de Negocio): Crearemos la función CambiarEstado(id, estado). Además, modificaremos tu función Procesar en el PlanillaService para inyectar una regla de oro: Si la planilla está CERRADA, el motor de cálculo aborta inmediatamente devolviendo un error.

Paso 3 (La UI): En la vista de "Detalle de Planilla", agregaremos un botón de "🔒 Cerrar Planilla" (con una advertencia fuerte). Si la planilla ya está cerrada, ocultaremos el botón de "⚙️ Procesar Cálculo" y el botón de "🗑️ Eliminar".