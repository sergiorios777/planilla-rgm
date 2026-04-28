# Plan de Implementación: Generación de Documentos PDF
## Meta 1: Preparación de Datos (Repositorio)
Actualmente, tu función `ObtenerDetalles` trae los totales (Ingresos, Retenciones, etc.). Para imprimir el reporte detallado y las boletas, necesitamos saber de qué están compuestos esos totales.

Acción: Crearemos una nueva función en `planilla_repository.go` llamada `ObtenerBoletasCompletas` que traiga la cabecera del detalle y, además, todas las filas de `planilla_conceptos` (Sueldo, EsSalud, Renta 5ta, AFP, etc.) asociadas a ese trabajador en esa planilla.

También necesitaremos traer los datos de la Municipalidad (Tenant) para la cabecera (Nombre, RUC).

## Meta 2: Integrar la Librería de PDF
No reinventaremos la rueda. Usaremos la librería estándar de la industria en Go para PDFs: `github.com/jung-kurt/gofpdf`. Es rápida, no requiere instalar dependencias externas en el servidor (como `wkhtmltopdf`) y dibuja directamente en binario.

Acción: Ejecutarás `go get github.com/jung-kurt/gofpdf` en tu terminal.

Crearemos un nuevo servicio: `internal/services/pdf_service.go`.

## Meta 3: Generador del "Reporte de Planilla" (El Resumen General)
Este es el documento que firma el Alcalde o el Gerente de RRHH.

Formato: A4 Horizontal (Landscape). Al tener muchas columnas (DNI, Nombre, Cargo, Sueldo, Asignación, Faltas, AFP, 5ta, Neto, Aportes), el formato horizontal es obligatorio para que sea legible.

Estructura:

Cabecera: Nombre de la Municipalidad, "Reporte de Planilla Mensual", Periodo y Descripción.

Tabla central: Iteración de todos los trabajadores.

Pie de página: Fila de "Totales Generales" (Suma de todos los ingresos de la muni, total a transferir a AFPs, total neto a pagar al banco).

## Meta 4: Generador de "Boletas de Pago" (Impresión Masiva)
Este es el documento que se entrega al trabajador (o se envía por correo).

Formato: A4 Vertical (Portrait).

Diseño Inteligente: Imprimiremos 2 boletas por página (una en la mitad superior y otra en la mitad inferior, o la original y la copia para firma en la misma hoja).

Contenido de la Boleta:

Datos del empleador (Municipalidad).

Datos del trabajador (DNI, Nombre, Régimen, Cargo).

Dos columnas paralelas: "Ingresos" (Izquierda) y "Retenciones/Descuentos" (Derecha).

Un cuadro inferior con el "Neto a Pagar" y los "Aportes del Empleador" (EsSalud).

Concatenación: Un solo archivo PDF contendrá las boletas de todos los trabajadores de esa planilla. Si hay 100 trabajadores, el PDF tendrá 50 páginas (a 2 por hoja). El usuario solo hace clic en "Imprimir" una vez.

## Meta 5: Enrutamiento y UI
Añadiremos los botones en tu vista `planilla_detalle_ui.html`.

El Truco UI: Para descargar PDFs, no usaremos HTMX. Usaremos enlaces HTML tradicionales (`<a href="..." target="_blank">`) estilizados como botones. ¿Por qué? Porque HTMX espera fragmentos HTML para inyectar en la página; si le mandas un PDF binario, no sabrá qué hacer. El `<a target="_blank">` le dice al navegador "Abre este PDF en una pestaña nueva", lo cual da una experiencia de usuario perfecta.