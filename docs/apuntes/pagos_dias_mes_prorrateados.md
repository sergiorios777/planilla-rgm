# Pagos de días prorrateados para trabajadores cesados - análisis

## Ideas iniciales
Al analiza la función procesar de @[internal/services/planilla_service.go] para saber si al procesar la planilla se tiene en cuenta aquellos contratos que ya no estan activos, por haber finalizado antes de la ejecución de la planilla, pero después del inicio del mes vigente. Encontramos esto:

Tras analizar el flujo de procesamiento de la planilla, la respuesta concreta es no, no se tienen en cuenta. Los contratos que ya no están activos (con  c.activo = false ) son excluidos por completo del cálculo de la planilla, incluso si finalizaron después del inicio del mes vigente.

Aquí te detallo la justificación técnica de este comportamiento basada en el código:

### 1. Extracción de contratos en el servicio

En el método  Procesar  de planilla_service.go, se obtienen los contratos a procesar llamando a:

    contratos, _ := s.Repo.ObtenerContratosActivosPlanilla(tenantID, anio, mes)

### 2. Filtro excluyente en el repositorio

En planilla_repository.go, la consulta SQL utilizada en  ObtenerContratosActivosPlanilla  contiene la siguiente condición restrictiva en su cláusula
WHERE :

    WHERE c.tenant_id = $1 AND c.activo = true
      AND c.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
      AND (c.fecha_fin IS NULL OR c.fecha_fin >= make_date($2, $3, 1)::date)
      AND p.es_dietario = false

* Como ves en la línea 116 de planilla_repository.go, se exige de forma estricta que  c.activo = true .
* Al dar de baja a un contrato (inactivarlo), su columna  activo  se actualiza a  false . Por ende, esta consulta lo omitirá totalmente, sin importar que su fecha de finalización ( c.fecha_fin ) sea posterior al primer día del mes de la planilla.

### Consecuencia:

Aunque la función  calcularDiasLaborados  del motor de cálculos de planillas en planilla_service.go está diseñada para prorratear los días de pago de contratos que finalizan a mitad de mes (utilizando  c.fecha_fin ), esta lógica nunca llega a ejecutarse para contratos dados de baja antes del cierre de planilla, ya que son pre-filtrados y descartados en la consulta del repositorio.
---

## Análisis de posibles soluciones

Identificamos un escenario clásico y muy importante en la gestión de nóminas: **el pago de fracciones de mes para trabajadores cesados**. Es correcto, actualmente, el sistema descarta a estos trabajadores porque la consulta exige estrictamente que el contrato esté activo (`c.activo = true`).

Por otro lado, se diseñó un motor de cálculo en Go. La función `calcularDiasLaborados` en tu archivo `planilla_service.go` ya contiene la lógica matemática para prorratear los días laborados basándose en la fecha de fin (`c.fecha_fin`). Por lo tanto, el motor ya está preparado; solo necesitamos "abrirle la puerta" en la base de datos a estos contratos específicos.

A continuación, presento el análisis de las estrategias y la solución más eficiente para implementar.

### Estrategias

Existen dos estrategias principales para abordar este escenario en el diseño de software de planillas:

* **Estrategia 1: Inclusión Dinámica en SQL (La más eficiente y recomendada).** Consiste en modificar la cláusula `WHERE` del repositorio para permitir el ingreso de contratos inactivos **únicamente** si su fecha de cese (`c.fecha_fin`) pertenece al mes y año de la planilla que se está procesando. Al hacerlo a nivel de base de datos, el consumo de memoria en Go sigue siendo mínimo y reutilizamos tu motor de prorrateo actual.


* **Estrategia 2: Módulo de Liquidaciones Independiente.** Algunas instituciones prefieren procesar a los trabajadores cesados en un lote separado llamado "Liquidación de Beneficios". Aunque es válido, rompe con el pago regular de sus días trabajados en la planilla mensual.

Implementaremos la **Estrategia 1** por ser la más rápida, eficiente y acorde a tu arquitectura actual.

---

### Código e Instrucciones de Implementación

Vamos a modificar el método `ObtenerContratosActivosPlanilla` dentro de tu repositorio.

#### Paso 1: Actualizar la consulta en el Repositorio

Abre el archivo `internal/repository/planilla_repository.go` y ubica el método `ObtenerContratosActivosPlanilla`. Debes reemplazar la consulta SQL (`query`) actual por la siguiente versión mejorada:

```go
func (r *PlanillaRepository) ObtenerContratosActivosPlanilla(tenantID int, anio int, mes int) ([]models.ContratoPlanilla, error) {
	query := `
	SELECT c.id, c.puesto_id, rl.codigo, 
		COALESCE(t.regimen_pensionario, 'ONP'), COALESCE(t.afp_id, 0), COALESCE(t.afp_tipo_comision, ''),
		c.fecha_inicio, c.fecha_fin,
		t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre_completo,
		t.numero_documento AS trabajador_numero_documento,
		p.nombre AS puesto_nombre,
		COALESCE(p.codigo_airhsp, '') AS puesto_codigo_airhsp,
		COALESCE(o.documento_aprobacion, 'N/A') AS organigrama_documento_aprobacion,
		COALESCE(uo.nombre, 'Sin Unidad') AS unidad_organica_nombre,
		COALESCE(uo.tipo, 'N/A') AS unidad_organica_tipo,
		p.sueldo_presupuestado AS sueldo_basico_historico
	FROM contratos c
	INNER JOIN trabajadores t ON c.trabajador_id = t.id
	INNER JOIN puestos p ON c.puesto_id = p.id
	INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
	LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
	LEFT JOIN organigramas o ON uo.organigrama_id = o.id
	WHERE c.tenant_id = $1 
	  AND p.es_dietario = false
	  AND c.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
	  AND (
	      -- Condición A: El contrato sigue activo (y su fecha fin, si existe, es futura o del mes actual)
	      (c.activo = true AND (c.fecha_fin IS NULL OR c.fecha_fin >= make_date($2, $3, 1)::date))
	      OR 
	      -- Condición B: El contrato está inactivo, PERO su fecha de fin ocurrió dentro de este mismo mes
	      (c.activo = false AND c.fecha_fin IS NOT NULL 
	       AND c.fecha_fin >= make_date($2, $3, 1)::date 
	       AND c.fecha_fin <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date)
	  )
	`
	rows, err := r.db.Query(query, tenantID, anio, mes)
    // ... el resto del código se mantiene exactamente igual[cite: 3]

```

### ¿Por qué funciona esta mejora?

1. **Condición A (`c.activo = true`)**: Mantiene el comportamiento habitual para todos los trabajadores regulares de la municipalidad.
2. **Condición B (`c.activo = false`)**: Actúa como un "salvoconducto". Si el contrato fue dado de baja, PostgreSQL revisará su `fecha_fin`. Si esta fecha cae exactamente dentro del mes que se está procesando (ej. 15 de mayo), el trabajador será incluido en la lista.


3. **El Motor de Go hace el resto**: Al llegar a `planilla_service.go`, la función `calcularDiasLaborados` detectará que `fechaFin` no es nulo, calculará correctamente los días proporcionales (ej. 15 días) y asignará el sueldo fraccionado de forma automática.

Con este simple pero poderoso cambio en la consulta SQL, tu sistema será capaz de procesar ceses a mitad de mes sin alterar la lógica de concurrencia ni las reglas de cálculo en Go.
---

## Comentario sobre la eficiencia de la consulta SQL

Aquí te explico por qué, paso a paso:

1. **El tamaño de los datos:** Para una base de datos robusta como PostgreSQL, procesar 1,000, 10,000 o incluso 100,000 registros es una tarea minúscula. Las bases de datos relacionales están diseñadas matemáticamente para buscar y filtrar datos en milisegundos.
2. **Evaluación de la consulta (Query Execution):** Cuando PostgreSQL lee la cláusula `WHERE`, utiliza algoritmos optimizados. Al agregar la condición `OR` para buscar contratos inactivos cuya fecha de fin esté en el mes actual, la base de datos simplemente evalúa las fechas. Para 1,000 trabajadores, esto tomará fracciones de segundo.
3. **El impacto en Go:** Tu servidor en Go recibirá la lista de 1,000 trabajadores (los activos) más esos 2 o 3 trabajadores cesados en el mes. Para Go, iterar sobre 1,003 elementos en lugar de 1,000 no tiene absolutamente ningún impacto perceptible en la memoria o en la velocidad de la concurrencia (goroutines).

### Resumen de la Solución

Actualizaremos la consulta SQL en el archivo del repositorio para que, además de buscar los contratos activos, permita ingresar aquellos contratos inactivos (`c.activo = false`) cuya fecha de finalización (`c.fecha_fin`) esté exactamente dentro del mes y año que la planilla está procesando. El resto del código de cálculo en Go se encargará automáticamente del prorrateo usando las fechas.

### Instrucciones de Implementación para el Agente Antigravity CLI

Aquí tienes las instrucciones detalladas y claras. Puedes copiar este bloque de texto y pegarlo directamente en tu terminal para que el agente ejecute el código.

---

**Instrucciones para Antigravity CLI:**

1. Abre el archivo `internal/repository/planilla_repository.go`.


2. Busca el método llamado `ObtenerContratosActivosPlanilla`.


3. Reemplaza **únicamente** la variable `query` de ese método con el siguiente código SQL actualizado:

```go
	query := `
	SELECT c.id, c.puesto_id, rl.codigo, 
		COALESCE(t.regimen_pensionario, 'ONP'), COALESCE(t.afp_id, 0), COALESCE(t.afp_tipo_comision, ''),
		c.fecha_inicio, c.fecha_fin,
		t.apellido_paterno || ' ' || t.apellido_materno || ', ' || t.nombres AS trabajador_nombre_completo,
		t.numero_documento AS trabajador_numero_documento,
		p.nombre AS puesto_nombre,
		COALESCE(p.codigo_airhsp, '') AS puesto_codigo_airhsp,
		COALESCE(o.documento_aprobacion, 'N/A') AS organigrama_documento_aprobacion,
		COALESCE(uo.nombre, 'Sin Unidad') AS unidad_organica_nombre,
		COALESCE(uo.tipo, 'N/A') AS unidad_organica_tipo,
		p.sueldo_presupuestado AS sueldo_basico_historico
	FROM contratos c
	INNER JOIN trabajadores t ON c.trabajador_id = t.id
	INNER JOIN puestos p ON c.puesto_id = p.id
	INNER JOIN regimenes_laborales rl ON p.regimen_id = rl.id
	LEFT JOIN unidades_organicas uo ON p.unidad_organica_id = uo.id
	LEFT JOIN organigramas o ON uo.organigrama_id = o.id
	WHERE c.tenant_id = $1 
	  AND p.es_dietario = false
	  AND c.fecha_inicio <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date
	  AND (
	      -- Condición A: El contrato sigue activo (y su fecha fin, si existe, es futura o del mes actual)
	      (c.activo = true AND (c.fecha_fin IS NULL OR c.fecha_fin >= make_date($2, $3, 1)::date))
	      OR 
	      -- Condición B: El contrato está inactivo, PERO su fecha de fin ocurrió dentro de este mismo mes
	      (c.activo = false AND c.fecha_fin IS NOT NULL 
	       AND c.fecha_fin >= make_date($2, $3, 1)::date 
	       AND c.fecha_fin <= (make_date($2, $3, 1) + interval '1 month' - interval '1 day')::date)
	  )
	`

```

4. Guarda el archivo y asegúrate de no modificar el bloque `rows, err := r.db.Query(query, tenantID, anio, mes)` ni el resto de la función.