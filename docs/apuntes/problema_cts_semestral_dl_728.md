# EL PROBLEMA:
Selección imprecisa de conceptos de la estructura de costos de un puesto para el cálculo de CTS semestral.

## FUNCIÓN CLAVE DEL PROCESO:
En la función 'func (s *CtsService) ProcesarCtsSemestral(tenantID int, anio int, periodo string) (int, error)' de 'internal\services\cts_service.go', se obtienen los montos que serán procesados durante el cálculo de la CTS semestral, las variables con estos valores son:

- contratos, err := s.Repo.ObtenerContratosCtsEligibles(tenantID, desde, hasta), específicamente la columna 'SueldoBasicoHistorico'.
- familiar, _ := s.Repo.ObtenerRemuneracionFamiliarActiva(c.PuestoID).
- grati, _ = s.Repo.ObtenerGratificacionHistorica(c.ID, anio|anio-1, 7|12).
- vars, err := s.Repo.ObtenerVariablesSemestre(c.ID, anio1, meses1, anio2, meses2).

## DETALLES OBSERVADOS:
1. La obtención de la remuneración principal (sueldo básico) se hace a partir de la columna 'SueldoBasicoHistorico' de la variable 'contratos', sin embargo, este valor procede específicamente de 'puestos.sueldo_presupuestado' y es un valor de referencia que puede variar al establecerse el valor 'oficial' en 'puesto_conceptos.monto' filtrada por el 'concepto_tenant_id' y el 'puesto_id', desde luego.
El 'concepto_tenant_id' se especifica de manera indirecta en 'var ConceptosMestrosCTS = map[string]map[string][]string' de 'internal\config\plantillas_conceptos.go' para el régimen laboral 'DL 728' en la clave 'remuneracion', el valor es el código interno (de los conceptos maestros) a partir del cual se puede obtener el 'concepto_id' relacionado en la tabla 'conceptos_tenant'. En el futuro se pueden agregar más códigos internos relacionados o variarlos.
2. La obtención de los montos del concepto de asignación familiar (código interno: ASIG_FAM_DL728) aunque es correcto, no hace referencia a 'ConceptosMestrosCTS' régimen laboral 'DL 728' clave 'asignacion_familiar'. Existe un riesgo en el 'harcodero' actual si la clave interna cambia, es más sencillo actualizar su valor en 'config.ConceptosMestrosCTS'.
3. La obtención de los montos históricos del concepto 'gratificacion' aunque es correcto, no referencia a 'ConceptosMestrosCTS' régimen laboral 'DL 728' clave 'gratificacion' hay varios valores para esta clave. Existe un riesgo en el 'harcodero' actual si las claves internas cambian, es más sencillo actualizar sus valores en 'config.ConceptosMestrosCTS'.
4. La obtención de 'vars' hace la consulta en las tablas correctas pero falla al no excluir del conjunto de conceptos resultante aquellos que son de la remuneración principal (sueldo básico) y asignación familiar, que también cumplen las condiciones en WHERE de la función del repositorio y están debidamente especificados en 'ConceptosMestrosCTS'.

## SUGERENCIAS PARA ANALIZAR:
- Basar la identificación de la remuneración principal, asignación familiar, gratificación y remuneraciones variables y otros para el cálculo de la CTS Semestral a partir de los conceptos (conceptos_tenant) identificados mediante el código interno (conceptos_maestros) especificados en 'Config.ConceptosMestrosCTS'.

## INFORMACIÓN RELEVANTE DE LAS TABLAS RELACIONADAS AL PROBLEMA:

### Tabla «public.puesto_conceptos»
      Columna       |     Tipo      | Ordenamiento | Nulable  |                 Por omisión
--------------------+---------------+--------------+----------+----------------------------------------------
 id                 | integer       |              | not null | nextval('puesto_conceptos_id_seq'::regclass)
 puesto_id          | integer       |              | not null |
 concepto_tenant_id | integer       |              | not null |
 monto              | numeric(10,2) |              |          |
 activo             | boolean       |              |          | true
Índices:
    "puesto_conceptos_pkey" PRIMARY KEY, btree (id)
    "idx_puesto_conceptos_pid" btree (puesto_id)
    "unique_puesto_concepto" UNIQUE CONSTRAINT, btree (puesto_id, concepto_tenant_id)
Restricciones de llave foránea:
    "puesto_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE
    "puesto_conceptos_puesto_id_fkey" FOREIGN KEY (puesto_id) REFERENCES puestos(id) ON DELETE CASCADE


 ### Tabla «public.conceptos_tenant»
           Columna           |            Tipo             | Ordenamiento | Nulable  |                   Por omisión
-----------------------------+-----------------------------+--------------+----------+-------------------------------------------------
 id                          | integer                     |              | not null | nextval('conceptos_tenant_id_seq'::regclass)
 tenant_id                   | integer                     |              | not null |
 concepto_id                 | integer                     |              | not null |
 nombre_personalizado        | character varying(150)      |              | not null |
 frecuencia_meses            | character varying(50)       |              |          | '1,2,3,4,5,6,7,8,9,10,11,12'::character varying
 activo                      | boolean                     |              |          | true
 clasificador_id             | integer                     |              |          |
 es_extraordinario           | boolean                     |              |          | false
 created_at                  | timestamp without time zone |              |          | CURRENT_TIMESTAMP
 updated_at                  | timestamp without time zone |              |          | CURRENT_TIMESTAMP
 requiere_monto              | boolean                     |              |          | false
 modelo_id                   | integer                     |              |          |
 es_pensionable              | boolean                     |              |          | false
 es_remunerativa             | boolean                     |              |          | false
 es_base_cts                 | boolean                     |              |          | false
 es_base_beneficios_sociales | boolean                     |              |          | false
 es_ocasional                | boolean                     |              | not null | false
 es_afecto_cargas_sociales   | boolean                     |              | not null | false
Índices:
    "conceptos_tenant_pkey" PRIMARY KEY, btree (id)
    "idx_conceptos_tenant_tid" btree (tenant_id)
    "unique_tenant_modelo_id" UNIQUE CONSTRAINT, btree (tenant_id, modelo_id)
Restricciones de llave foránea:
    "conceptos_tenant_clasificador_id_fkey" FOREIGN KEY (clasificador_id) REFERENCES clasificadores_mef(id) ON DELETE SET NULL
    "conceptos_tenant_concepto_id_fkey" FOREIGN KEY (concepto_id) REFERENCES conceptos_maestros(id)
    "conceptos_tenant_modelo_id_fkey" FOREIGN KEY (modelo_id) REFERENCES conceptos_modelo(id) ON DELETE CASCADE
    "conceptos_tenant_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
Referenciada por:
    TABLE "planilla_conceptos" CONSTRAINT "planilla_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE SET NULL
    TABLE "puesto_conceptos" CONSTRAINT "puesto_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE
    TABLE "regimen_concepto_tenant" CONSTRAINT "regimen_concepto_tenant_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE

### Tabla «public.conceptos_maestros»
    Columna     |           Tipo           | Ordenamiento | Nulable  |                  Por omisión
----------------+--------------------------+--------------+----------+------------------------------------------------
 id             | integer                  |              | not null | nextval('conceptos_maestros_id_seq'::regclass)
 codigo         | character varying(50)    |              | not null |
 descripcion    | character varying(255)   |              | not null |
 tipo           | character varying(50)    |              | not null |
 activo         | boolean                  |              |          | true
 created_at     | timestamp with time zone |              |          | CURRENT_TIMESTAMP
 parent_id      | integer                  |              |          |
 codigo_interno | character varying(50)    |              | not null |
 origen         | character varying(20)    |              | not null | 'sunat'::character varying
Índices:
    "conceptos_maestros_pkey" PRIMARY KEY, btree (id)
    "conceptos_maestros_codigo_interno_key" UNIQUE CONSTRAINT, btree (codigo_interno)
Restricciones CHECK:
    "chk_conceptos_maestros_origen" CHECK (origen::text = ANY (ARRAY['sunat'::character varying, 'interno'::character varying]::text[]))
Restricciones de llave foránea:
    "conceptos_maestros_parent_id_fkey" FOREIGN KEY (parent_id) REFERENCES conceptos_maestros(id) ON DELETE SET NULL
Referenciada por:
    TABLE "conceptos_afectaciones" CONSTRAINT "conceptos_afectaciones_concepto_base_id_fkey" FOREIGN KEY (concepto_base_id) REFERENCES conceptos_maestros(id) ON DELETE CASCADE
    TABLE "conceptos_afectaciones" CONSTRAINT "conceptos_afectaciones_concepto_derivado_id_fkey" FOREIGN KEY (concepto_derivado_id) REFERENCES conceptos_maestros(id) ON DELETE CASCADE
    TABLE "conceptos_maestros" CONSTRAINT "conceptos_maestros_parent_id_fkey" FOREIGN KEY (parent_id) REFERENCES conceptos_maestros(id) ON DELETE SET NULL
    TABLE "conceptos_modelo" CONSTRAINT "conceptos_modelo_concepto_id_fkey" FOREIGN KEY (concepto_id) REFERENCES conceptos_maestros(id)
    TABLE "conceptos_tenant" CONSTRAINT "conceptos_tenant_concepto_id_fkey" FOREIGN KEY (concepto_id) REFERENCES conceptos_maestros(id)