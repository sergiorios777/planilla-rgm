## Las planillas adicionales:

### ¿Por qué ocurren?
Son generadas cuando un usuario tiene:
- Algunas normas legales que exigen su creación por disposiciones específicas, con determinados conceptos no extraordinarios (ej. gratificaciones CAS)
- Una fecha de inicio posterior al cierre de una planilla ordinaria (por ejemplo, 15 de junio).
- Una fecha de fin posterior al cierre de la planilla ordinaria.
- Omisión de algún trabajador en la planilla ordinaria.
- Omisión de algún concepto en la planilla ordinaria.
- Porque se requiere separar algunos conceptos de la planilla ordinaria por una razón específica (ej. pactos colectivos institucionales, bonos, etc.)

### El problema no es solo generar planillas adicionales, existe la 'Composición de conceptos'
Los operadores de recursos humanos varían sus criterios de composición de conceptos, tanto en planillas ordinarias como adicionales. Por ejemplo:
- Por costumbre administrativa incluyen pactos colectivos o bonos institucionales en las planillas ordinarias.
- Otros deciden incluirlos en planillas adicionales.
- Algunos separan conceptos remunerativos exclusivamente en planillas ordinarias y conceptos no remunerativos en planillas adicionales.
- Otros aún no han definido el momento en el que se deben generar estas planillas.
Nuestra aplicación sugiere una composición estándar, pero debe permitir la modificación de la misma para adecuarse a las necesidades de cada municipalidad.

### ¿Son diferentes las planillas ordinarias de las planillas adicionales?

Tecnicamente no son diferentes en cuanto a su estructura, ya que ambas responden a la misma lógica de:

- tener un periodo de inicio y fin
- sumar los conceptos de ingresos (remuneraciones, pensiones, etc.)
- restar los conceptos de egresos (detracciones, descuentos, etc.)
- sumar los conceptos de aportes del empleador cuando corresponda.
- obtener el total del trabajador y el aporte del empleador.
- comparten conceptos, periodos y trabajadores de las planillas ordinarias.
- Son calculadas utilizando los mismos procesos de las planillas ordinarias.
- No requiere la creación de tablas adicionales (con excepción quizá de algunas temporales).

Sin embargo, son diferentes en cuanto a su:

- propósito: las planillas ordinarias se generan para pagar a los trabajadores en un periodo determinado, mientras que las planillas adicionales se generan para corregir errores u omisiones en las planillas ordinarias.
- nombre: las planillas adicionales se identifican con el sufijo "(ADIC)" para diferenciarlas de las planillas ordinarias.
- fecha de emisión: las planillas adicionales se generan con una fecha de emisión posterior a la fecha de cierre de la planilla ordinaria, en casos de corrección de errores u omisiones. Puede coincidir con la fecha de cierre de la planilla ordinaria en casos de nuevos ingresos, egresos u otros conceptos no programados. Sin embargo, se recomienda que sea posterior para evitar confusiones.
- fecha de cierre: las planillas adicionales se cierran con una fecha de cierre posterior a la fecha de cierre de la planilla ordinaria. Puede coincidir con la fecha de cierre de la planilla ordinaria en casos de nuevos ingresos, egresos u otros conceptos no programados. Sin embargo, se recomienda que sea posterior para evitar confusiones.

### Funciones que nos faltaría agregar para lograr la versatilidad que tienen las planillas adicionales

1. **Capacidad para seleccionar conceptos a procesar en las planillas ordinarias o adicionales:**
   * El sistema debe permitir seleccionar los conceptos que se van a procesar en las planillas ordinarias o adicionales. Para logarlo debemos modificar la estructura de las tablas `conceptos_tenant` y su hermano mayor `conceptos_modelo` para incluir una columna que permita indicar si el concepto se debe procesar por defecto en una planilla ordinaria, la podemos llamar `procesar`.
   * Esta modificación nos permitiría procesar las planillas ordinarias sin ninguna selección adicional conceptos, basados en la que nosotros previmos en la configuración de cada concepto.
   * Podemos aprovechar que las planillas ordinarias y las adicionales son iguales en estructura para generar planillas 'adicionales o especiales' seleccionando manualmente su composición de comceptos.
   * ¿cómo guardamos el estado (`procesar`) de los conceptos para cada planilla ordinaria o adicional? No creo que sea buena idea modificar la tabla `conceptos_tenant`, por lo que se me ocurre que podemos crear una tabla temporal `planilla_conceptos_seleccionados` que guarde el estado de los conceptos de cada planilla hasta que el estado de la planilla (`planillas`) sea 'CERRADO'.
   * En caso de que exista un estado 'CERRADO' para una planilla, no se debe permitir modificar la selección de conceptos.
   * En la vista `@ui\templates\tenant\planillas_ui.html`, en la fila de la planilla creada, debemos agregar un 'botón' que llame a un modal que permita seleccionar los conceptos a procesar (quizá utilizando un `checkbox` con `role="switch"` para cada concepto). Este modal debe tener la capacidad de buscar por nombre o descripción del concepto y quizá algunos botones de filtro (estilo `pico group`) de características de los conceptos (remunerativo, ordinario, etc.)
   * Los conceptos deben obtenerse de los conceptos registrados en la tabla `puesto_conceptos` a la cual podemos acceder seleccionando los contratos vigentes (tabla `contratos`) a la fecha de procesar la planilla, en la cual tenemos el campo `puesto_id` que comparte con la tabla `puesto_conceptos`.
   * El riesgo de permitir la selección de conceptos para las planillas ordinarias y adicionales es **muy alta**, una selección descuidada puede causar un gran desastre. El riesgo debe ser asumido por el operador de recursos humanos de la municipalidad. **Debemos incluir un mensaje de advertencia en la UI para alertar sobre esta situación.**

2. **Capacidad de seleccionar trabajadores específicos para planillas adicionales (inclusive permitir la exclusión de trabajadores de las planillas ordinarias):**
   * El sistema debe permitir seleccionar trabajadores para incluirlos en una planilla adicional o excluirlos de las planillas ordinarias, esto sí puede afectar la manera cómo se procesa la planilla ordinaria (se procesa a todos los trabajadores activos de una sola vez), por lo que tenemos que tener cuidado de modificar la lógica de procesamiento de las planillas, analizando en qué punto puede hacerce este cambio sin afectar la mayor parte de la lógica que ya funciona, así podremos aprovechar el procesamiento para las planillas ordinarias y las adicionales.
   * Quizá sea conveniente que la tabla `planillas` tenga una columna `procesar_todos_trabajadores` que almacene el `TRUE` (valor por defecto) en caso de que la planilla tenga todos los trabajadores y `FALSE` en caso de que no los tenga. En caso de `FALSE` debemos tener una forma de saber que trabajadores fueron excluidos, los demás se entiende que fueron incluidos en el procesamiento de la planilla. Por el momento no tengo una idea clara de cómo lograr esto, quizá una columna tipo snapshot con el id o DNI de los trabajadores excluidos.
   * En la vista `@ui\templates\tenant\planillas_ui.html`, en la fila de la planilla creada, debemos agregar un 'botón' que llame a un modal que permita seleccionar los trabajadores a incluir o excluir de la planilla.
   * El modal debe mostrar los trabajadores contrato vigente a la fecha de proceso de la planilla, permitiendo su selección individual y por regimen laboral (usando los filtros) y un buscador.
   * Debemos incluir un mensaje de advertencia en la UI para alertar sobre la situación de selección individual de los trabajadores.
   * Una idea para mantener la persistencia de los trabajadores seleccionados es tener una tabla temporal `planilla_trabajadores_seleccionados` que guarde el estado de los trabajadores de cada planilla hasta que el estado de la planilla (`planillas`) sea 'CERRADO'.
   * En caso de que exista un estado 'CERRADO' para una planilla, no se debe permitir modificar la selección de trabajadores.
   * El riesgo de permitir la selección de trabjadores para las planillas ordinarias y adicionales es **muy alta**, debe ser asumida por el operador de recursos humanos de la municipalidad. **Debemos incluir un mensaje de advertencia en la UI para alertar sobre esta situación.**
   
   
### Creación de compoenetes para la UI
- Podemos aprovechar y crear componentes reusables para la selección de conceptos y trabajadores. Toma inspiración del componente `@ui\templates\components\buscador_codigo_sunat_modal.html`.

### Modificación más profunda de la vista para procesar planillas ordinarias y adicionales
- Debido a que puede existir la posibilidad de selección de conceptos y trabajadores por cada planilla que se va a procesar, quizá en la vista `@ui\templates\tenant\planillas_ui.html` en lugar del botón 'Procesar planilla' debamos poner un botón de 'Planificar' o 'Configurar' o algo similar (ya no tendría los botones para acceder a los modales de selección de conceptos y trabajadores en esta vista). Este botón nos llevará a una **vista de planificación/configuración** de la planilla en la cual, podremos seleccionar los conceptos y trabajadores que se van a procesar en la planilla, a través de los modales que he mencionado anteriormente.
- Esta vista debe mostrar la planilla que se va a procesar y los conceptos y trabajadores seleccionados, con la posibilidad de modificar la selección. Si no es una práctica inadecuada podemos tener dos bloques paginados (para evitar listas interminables) en la vista, con buscadores y filtros, uno para los conceptos y otro para los trabajadores.
- Debemos incluir un botón de 'Procesar planilla' o algo similar, y mientras la planilla no esté en estado 'CERRADO' se puede regresar a esta vista, caso contrario ya no se podrá regresar, sino solo acceder a la vista de detalles de la planilla.
