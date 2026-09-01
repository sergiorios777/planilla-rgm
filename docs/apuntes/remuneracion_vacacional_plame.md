# Sobre la codificación PLAME/SUNAT de remuneración vacacional

## Contexto:

### El problema:
**¿Cómo manejamos la clasificación de mediante código sunat (para PLAME) a partir de la tabla `planilla_conceptos`?**
- La información de las planillas se guardan en las tablas `planillas`, `planilla_detalles` (guarda el detalle de los trabajadores) y `planilla_conceptos` (guardan los conceptos detallados de las planillas, ingresos, retenciones y aportes).
- La tabla `personal_licencias_vacaciones` tiene la información de vacaciones o licencias por cada trabajador, incluyendo las fechas de inicio y fin de la suspensión del trabajo regular.

### Precisiones:
- La tabla `conceptos_tenant` tienen una columna con el flag `es_remunerativa`.
- La clasificación de la remuneración es requerida solo para la declaración del PLAME, que debe identificar el monto del mes correspondiente a remuneración vacacional, el trabajador pueda salir de vacaciones o de licencia dentro de cualquier día del mes, por lo que existe una parte del mes que trabajó y otra que tenía las vacaciones o licencia.
- Hay al menos 3 conceptos de ingresos de la tabla paramétrica 22 de sunat para asociar la remuneración vacaciones:
    
    | código |             descripción ingreso              | Régimen laboral |
    | ------ | -------------------------------------------- | --------------- |
    |  2007  | REMUNERACIÓN VACACIONAL                      | DL 276, DL 728  |
    |  2043  | REMUNERACIÓN VACACIONAL-D.LEG. 1057-CAS      | DL 1057         |
    |  2049  | ENTREGA ECONÓMICA POR VACACIONES - LEY 30057 | LEY 30057       |


### Ideas iniciales:
- Crear e insertar los conceptos de remuneración vacacional para los trabajadores con uso de vacaciones dentro del mes, es impreciso, pues los conceptos de ingresos no varían realmente, sino solamente en la declaración a sunat mediante el PLAME se debe identificar monto de los ingresos remunerativos para un periodo devengado, utilizando los códigos válidos para el PLAME (tabla 22 sunat).
- Podemos crear una tabla auxiliar que sea un espejo de `planilla_conceptos`, en el que se puedan segregar proporcionalmente los conceptos asociados al trabajador que se encuentra de vacaciones, pero ¿Cómo unimos la información de esta tabla que ya tendría los conceptos de la boleta dividios de forma proporcional por los códigos normales y el código de remuneración vacacional correspondiente? ¿Se puede unir las tablas y excluir el trabajador de la tabla `planilla_conceptos`?

### Modelos SQL de las tablas relevantes:

**planilla_rgm=# \d planillas**
                                            Tabla «public.planillas»
      Columna      |           Tipo           | Ordenamiento | Nulable  |              Por omisión
-------------------+--------------------------+--------------+----------+---------------------------------------
 id                | integer                  |              | not null | nextval('planillas_id_seq'::regclass)
 tenant_id         | integer                  |              | not null |
 anio              | integer                  |              | not null |
 mes               | integer                  |              | not null |
 descripcion       | character varying(255)   |              | not null |
 estado            | character varying(20)    |              |          | 'BORRADOR'::character varying
 created_at        | timestamp with time zone |              |          | CURRENT_TIMESTAMP
 updated_at        | timestamp with time zone |              |          | CURRENT_TIMESTAMP
 es_extraordinaria | boolean                  |              | not null | false
 tipo              | character varying(30)    |              | not null | 'ORDINARIA'::character varying
Índices:
    "planillas_pkey" PRIMARY KEY, btree (id)
    "idx_planillas_tenant" btree (tenant_id, anio, mes)
    "idx_planillas_tenant_tipo" btree (tenant_id, tipo, anio, mes)
    "unique_planilla_mes" UNIQUE CONSTRAINT, btree (tenant_id, anio, mes, descripcion)
Restricciones de llave foránea:
    "planillas_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
Referenciada por:
    TABLE "planilla_detalles" CONSTRAINT "planilla_detalles_planilla_id_fkey" FOREIGN KEY (planilla_id) REFERENCES planillas(id) ON DELETE CASCADE
    TABLE "planilla_especial_conceptos" CONSTRAINT "planilla_especial_conceptos_planilla_id_fkey" FOREIGN KEY (planilla_id) REFERENCES planillas(id) ON DELETE CASCADE
    TABLE "planillas_cts" CONSTRAINT "planillas_cts_planilla_id_fkey" FOREIGN KEY (planilla_id) REFERENCES planillas(id) ON DELETE CASCADE


**planilla_rgm=# \d planilla_detalles**
                                                  Tabla «public.planilla_detalles»
             Columna              |          Tipo          | Ordenamiento | Nulable  |                  Por omisión
----------------------------------+------------------------+--------------+----------+-----------------------------------------------
 id                               | integer                |              | not null | nextval('planilla_detalles_id_seq'::regclass)
 planilla_id                      | integer                |              | not null |
 contrato_id                      | integer                |              | not null |
 total_ingresos                   | numeric(10,2)          |              |          | 0.00
 total_retenciones                | numeric(10,2)          |              |          | 0.00
 total_aportes                    | numeric(10,2)          |              |          | 0.00
 neto_pagar                       | numeric(10,2)          |              |          | 0.00
 trabajador_nombre_completo       | character varying(250) |              |          |
 trabajador_numero_documento      | character varying(20)  |              |          |
 puesto_codigo_airhsp             | character varying(50)  |              |          |
 puesto_nombre                    | character varying(200) |              |          |
 organigrama_documento_aprobacion | character varying(200) |              |          |
 unidad_organica_nombre           | character varying(200) |              |          |
 unidad_organica_tipo             | character varying(50)  |              |          |
 sueldo_basico_historico          | numeric(10,2)          |              |          |
Índices:
    "planilla_detalles_pkey" PRIMARY KEY, btree (id)
    "unique_detalle_contrato" UNIQUE CONSTRAINT, btree (planilla_id, contrato_id)
Restricciones de llave foránea:
    "planilla_detalles_contrato_id_fkey" FOREIGN KEY (contrato_id) REFERENCES contratos(id)
    "planilla_detalles_planilla_id_fkey" FOREIGN KEY (planilla_id) REFERENCES planillas(id) ON DELETE CASCADE
Referenciada por:
    TABLE "planilla_conceptos" CONSTRAINT "planilla_conceptos_planilla_detalle_id_fkey" FOREIGN KEY (planilla_detalle_id) REFERENCES planilla_detalles(id) ON DELETE CASCADE


**planilla_rgm=# \d planilla_conceptos**
                                            Tabla «public.planilla_conceptos»
       Columna       |          Tipo          | Ordenamiento | Nulable  |                  Por omisión
---------------------+------------------------+--------------+----------+------------------------------------------------
 id                  | integer                |              | not null | nextval('planilla_conceptos_id_seq'::regclass)
 planilla_detalle_id | integer                |              | not null |
 concepto_tenant_id  | integer                |              |          |
 tipo_concepto       | character varying(20)  |              | not null |
 monto               | numeric(10,2)          |              | not null |
 maestro_id          | integer                |              | not null | 0
 codigo_sunat        | character varying(10)  |              |          |
 nombre_en_boleta    | character varying(150) |              |          |
 fuente_rubro_id     | integer                |              |          |
 meta_id             | integer                |              |          |
Índices:
    "planilla_conceptos_pkey" PRIMARY KEY, btree (id)
    "idx_planilla_conceptos_meta" btree (meta_id)
    "idx_planilla_conceptos_rubro" btree (fuente_rubro_id)
Restricciones de llave foránea:
    "planilla_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE SET NULL
    "planilla_conceptos_fuente_rubro_id_fkey" FOREIGN KEY (fuente_rubro_id) REFERENCES fuentes_rubros(id)
    "planilla_conceptos_meta_id_fkey" FOREIGN KEY (meta_id) REFERENCES metas_presupuestales(id)
    "planilla_conceptos_planilla_detalle_id_fkey" FOREIGN KEY (planilla_detalle_id) REFERENCES planilla_detalles(id) ON DELETE CASCADE



**planilla_rgm=# \d conceptos_tenant**
                                                    Tabla «public.conceptos_tenant»
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
 modalidad_entrega           | character varying(20)       |              | not null | 'PERMANENTE'::character varying
 base_calculo_para           | text[]                      |              |          | '{}'::text[]
Índices:
    "conceptos_tenant_pkey" PRIMARY KEY, btree (id)
    "idx_conceptos_tenant_tid" btree (tenant_id)
    "unique_tenant_id_id" UNIQUE CONSTRAINT, btree (tenant_id, id)
    "unique_tenant_modelo_id" UNIQUE CONSTRAINT, btree (tenant_id, modelo_id)
Restricciones CHECK:
    "conceptos_tenant_modalidad_entrega_check" CHECK (modalidad_entrega::text = ANY (ARRAY['PERMANENTE'::character varying, 'PERIODICO'::character varying, 'EXCEPCIONAL'::character varying, 'OCASIONAL'::character varying]::text[]))
Restricciones de llave foránea:
    "conceptos_tenant_clasificador_id_fkey" FOREIGN KEY (clasificador_id) REFERENCES clasificadores_mef(id) ON DELETE SET NULL
    "conceptos_tenant_concepto_id_fkey" FOREIGN KEY (concepto_id) REFERENCES conceptos_maestros(id)
    "conceptos_tenant_modelo_id_fkey" FOREIGN KEY (modelo_id) REFERENCES conceptos_modelo(id) ON DELETE CASCADE
    "conceptos_tenant_tenant_id_fkey" FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
Referenciada por:
    TABLE "contrato_conceptos_snapshot" CONSTRAINT "contrato_conceptos_snapshot_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE
    TABLE "descuento_conceptos" CONSTRAINT "descuento_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE RESTRICT
    TABLE "descuentos" CONSTRAINT "descuentos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE RESTRICT
    TABLE "base_regimen_tenant" CONSTRAINT "fk_base_regimen_tenant_concepto" FOREIGN KEY (tenant_id, concepto_tenant_id) REFERENCES conceptos_tenant(tenant_id, id) ON DELETE CASCADE
    TABLE "planilla_conceptos" CONSTRAINT "planilla_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE SET NULL
    TABLE "planilla_especial_conceptos" CONSTRAINT "planilla_especial_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE
    TABLE "puesto_conceptos" CONSTRAINT "puesto_conceptos_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE
    TABLE "regimen_concepto_tenant" CONSTRAINT "regimen_concepto_tenant_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE
    TABLE "reglas_financiamiento_concepto" CONSTRAINT "reglas_financiamiento_concepto_concepto_tenant_id_fkey" FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id) ON DELETE CASCADE

---

## Respuesta (Chat Gemini Flas 3.7 extended):

El problema que describes es muy común al integrar la planilla operativa con la Planilla Electrónica (PLAME): para la institución y la boleta de pago, el sueldo mensual sigue siendo el mismo concepto (ej. haber básico), pero para SUNAT, los días de descanso vacacional efectivo deben declararse segregados bajo los códigos 2007, 2043 o 2049 según el régimen laboral.

---

### Análisis Crítico y Puntos Débiles del Planteamiento

* **Punto Débil 1: Crear una tabla espejo auxiliar en la BD.**
* *Por qué evitarlo:* Rompe el principio de *Única Fuente de Verdad* (SSOT). Una tabla física duplicada genera sobrecarga de almacenamiento, riesgo de desincronización si la planilla se recalcula en borrador, y obliga a crear lógica adicional de mantenimiento y borrado en cascada.


* *Realidad arquitectónica:* La segregación de remuneración vacacional para PLAME es un **problema de proyección/transformación de exportación**, no un cambio en la persistencia contable de la planilla.




* **Punto Débil 2: Intentar modificar o dividir filas dentro de `planilla_conceptos`.**
* Si divides los conceptos directamente en `planilla_conceptos`, afectarás la emisión de boletas, resúmenes presupuestales de rubros/metas y reportes históricos, duplicando filas de conceptos básicos de forma artificial.




* **Enfoque Correcto:**
Mantener `planilla_conceptos` intacta como reflejo fiel de la boleta y el presupuesto. La segregación proporcional debe resolverse **en memoria durante el proceso de exportación del PLAME** mediante un servicio especializado en Go (`plame_service.go`) o a través de una **Vista SQL de Proyección** que cruce `planilla_conceptos` con `personal_licencias_vacaciones`.



---

### Lógica de Transformación para la Exportación PLAME

$$\text{Monto Vacacional} = \sum (\text{Conceptos con } \texttt{es\_remunerativa = true}) \times \left(\frac{\text{Días Vacaciones Mes}}{30}\right)$$

$$\text{Monto Ordinario} = \text{Monto Total Concepto} - \text{Monto Vacacional}$$

* **Para conceptos ordinarios:** Se mantiene su `codigo_sunat` habitual con el monto neto de días laborados.


* **Para la porción vacacional:** Se consolida (o desglosa) y se le asigna el código de la Tabla 22 según el régimen laboral del contrato/puesto:


* D.L. 276 y D.L. 728 $\rightarrow$ Código **`2007`** (Remuneración Vacacional)


* D.L. 1057 (CAS) $\rightarrow$ Código **`2043`** (Remuneración Vacacional - CAS)


* Ley 30057 (Servir) $\rightarrow$ Código **`2049`** (Entrega Económica por Vacaciones)





---

### Solicitud Estructurada para Antigravity 2.0 CLI

```markdown
Actúa como Arquitecto de Software Go y Desarrollador Backend PostgreSQL.

### Contexto del Problema
Para la exportación mensual del archivo de importación al PDT/PLAME de SUNAT (archivos .rem y .jor), es necesario segregar los ingresos del trabajador cuando este ha gozado de vacaciones durante el mes:
1. La tabla `planilla_conceptos` contiene los conceptos remunerativos completos calculados para la boleta y presupuesto[cite: 3].
2. La tabla `personal_licencias_vacaciones` registra los periodos de descanso vacacional (fechas de inicio y fin)[cite: 3].
3. La porción de ingresos con `conceptos_tenant.es_remunerativa = true` correspondiente a los días de vacaciones debe reportarse en el archivo de PLAME bajo el código SUNAT correspondiente al régimen laboral del trabajador (2007 para 276/728, 2043 para CAS 1057, 2049 para Ley 30057), mientras que el saldo restante conserva sus códigos SUNAT habituales[cite: 3].

### Archivos de Referencia en el Workspace
1. Estructura de BD: `docs/temporal/estructura_planilla_rgm.sql`[cite: 3]
2. Entidades relevantes: `planillas`, `planilla_detalles`, `planilla_conceptos`, `conceptos_tenant`, `personal_licencias_vacaciones`, `contratos`, `puestos`, `regimenes_laborales`[cite: 3].

### Directriz Arquitectónica
NO crear tablas espejo físicas ni alterar la estructura persistida en `planilla_conceptos`[cite: 3]. La segregación debe resolverse mediante un servicio de transformación en Go (`PlameExportService`) o una consulta SQL de proyección específica para exportación[cite: 3].

### Tareas Requeridas
1. **Diseño del Algoritmo de Prorrateo en Go:**
   - Crear una función que reciba el `planilla_id` y el periodo (mes/año)[cite: 3].
   - Identificar trabajadores con registros activos en `personal_licencias_vacaciones` durante el mes de la planilla[cite: 3].
   - Calcular los días de vacaciones dentro del mes ($d_{vac}$) y los días efectivamente trabajados ($30 - d_{vac}$)[cite: 3].
   - Para cada concepto con `es_remunerativa = true`:
     * Generar la línea de ingreso ordinario por el factor $(30 - d_{vac}) / 30$ con su `codigo_sunat` original[cite: 3].
     * Generar la línea de remuneración vacacional por el factor $d_{vac} / 30$, asignando automáticamente el código SUNAT (2007, 2043, 2049) según el régimen laboral del puesto/contrato[cite: 3].
2. **Estructura del Servicio `internal/services/plame_service.go`:**
   - Implementar los métodos de obtención y formateo para generar las estructuras de datos listas para la descarga del archivo `.rem` de SUNAT.
3. **Casos Borde y Validaciones:**
   - Trabajador con mes completo de vacaciones (30 días): todo el monto remunerativo pasa al código vacacional[cite: 3].
   - Conceptos no remunerativos (`es_remunerativa = false`): no sufren prorrateo vacacional y se reportan al 100% con su código SUNAT configurado[cite: 3].

```