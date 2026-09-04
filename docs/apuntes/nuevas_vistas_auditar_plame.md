## [EL PROBLEMA]:
Los usuarios contadores deben llevar un control muy minucioso de la auditoría de códigos sunat. Por lo tanto, facilitar el acceso ágil al detalle bien sea buscando directamente en los códigos declarados como ya lo hacemos en `@ui\templates\tenant\planilla_sunat_codigos_ui.htm`, sin embargo, también se requiere la búsqueda rápida por:

- Trabajadores (nombre y DNI).
- Conceptos en boleta (descripción).

## [IDEAS]:
* **Accesos**: Cada una de las páginas directas a 'trabajadores' o 'conceptos en boleta' deberían tener su acceso desde la vista `@ui\templates\tenant\planilla_sunat_codigos_ui.htm`, con botones o enlaces directos a sus propias vistas (independientes).

* **Vista para `Trabajadores`**:
  - Ya tenemos una base excelente y funcional y es la vista `@ui\templates\tenant\plame_concepto_trabajadores_ui.html`, y su vista complementaria `@ui\templates\tenant\plame_trabajador_edicion_ui.html`.
  - El problema que tiene la vista `@ui\templates\tenant\plame_concepto_trabajadores_ui.html` es que actualmente solo funciona "filtrado" por el código sunat seleccionado en `@ui\templates\tenant\planilla_sunat_codigos_ui.htm`; y necesitamos que en su forma de vista independiente no tenga ningún filtro por el código sunat. Convendría que esta vista independiente (y quizá la filtrada) tenga paginación (según nuestra regla de diseño).

* **Vista para 'Conceptos en boleta'**:
  - Puede ser similar a la vista `@ui\templates\tenant\planilla_sunat_codigos_ui.htm`, pero en lugar de los conceptos maestros de sunat (tabla 22) debe mostrar los conceptos tenant que utiliza la planilla que se está auditando.
  - Dentro del bloque donde irá la tabla de los conceptos que debe consignar una columna para el Código SUNAT y un botón para mostrar un modal que permita modificar el Código SUNAT utilizando el componente 'buscador código sunat modal'. ¿Es factible o recomendable utilizar un modal sobre otro modal?
  - Los cambios de Código SUNAT efectuados afectan en bloque a todos los conceptos que se encuentran en `planilla_conceptos` o en la tabla que corresponda.

* **Consideraciones**:
  - Ten siempre sen consideración las reglas de diseño y arquitectura de la aplicación, para la paginación, para los botones, etc.