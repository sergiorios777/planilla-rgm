ANÁLISIS Y PROPUESTA DE IMPLEMENTACIÓN: DATA-DRIVEN CONCEPTOS CALCULADOS

Los análisis que has subido (`analisis_1.md`, `analisis_3.md` y el diagrama de base de datos) son **oro puro**. Quien sea (o la herramienta) que te haya ayudado a generar esos diagnósticos ha dado exactamente en el clavo.

Para sacarte de ese bloqueo, vamos a deconstruir este tema. Mi objetivo hoy es que **entiendas a la perfección el problema conceptual, por qué tu idea original tenía una pequeña falla estructural, y cómo la nueva estrategia de solución es la arquitectura definitiva para tu SaaS.**

---

### 1. Entendiendo el Problema Central: "La Fuga de Dominios"

En tu propuesta original, intentaste crear una "Tabla Maestra" (`conceptos_base_por_regimen`) que tenía estas columnas al mismo tiempo:

* `concepto_calculado_id` (Ej. CTS)
* `regimen_id` (Ej. DL 728)
* `concepto_modelo_id` (La plantilla global de tu SaaS)
* `concepto_tenant_id` (La copia local de la municipalidad)

**¿Cuál era el problema conceptual aquí?**
Estabas mezclando dos universos (dominios) que jamás deben tocarse directamente en la misma fila de una base de datos SaaS: **La Ley (Global)** y **La Realidad Operativa (Local)**.

* **La ambigüedad:** Si el motor de Go va a calcular la planilla de la Municipalidad de Iquitos... ¿Qué columna lee? ¿Lee el `modelo_id` o lee el `tenant_id`? Si un registro tiene ambos llenos, ¿cuál manda? Si tiene uno vacío, ¿el código debe hacer un `IF/ELSE`?
* **La sincronización imposible:** Si la Municipalidad decide que un bono específico NO entra en su cálculo de CTS (porque tienen un pacto colectivo especial), y tú borras esa fila, ¡estarías borrando la relación del `concepto_modelo_id` para TODOS los demás inquilinos de tu SaaS!

Los análisis 1 y 3 detectaron esto y te dijeron: *"Tu idea del motor basado en datos es genial, pero no almacenes simultáneamente el modelo y el tenant en la misma tabla"*.

---

### 2. La Estrategia de Solución: El Patrón "Siembra y Sobrescritura" (Seed & Override)

Para resolver esto, la estrategia arquitectónica que se plantea (y que está perfectamente graficada en el archivo `modelo_conceptos_calculados_refinado_analisis_1.html` que subiste) es **dividir esa tabla en dos**.

Separamos la responsabilidad en dos capas:

#### Capa A: El Catálogo Maestro (Lo que dice la Ley)

Creamos la tabla `base_regimen_default`.
Esta tabla es tuya, del **Súper Admin**. Aquí mapeas que la *CTS del Régimen 728* requiere el *Sueldo Base (Modelo)* y la *Asignación Familiar (Modelo)*.
Esta tabla **NO TIENE** `tenant_id`. Es universal.

#### Capa B: La Configuración Municipal (La Realidad del Cliente)

Creamos la tabla `base_regimen_tenant`.
Esta tabla le pertenece a cada municipalidad. Tiene `tenant_id` explícito y se relaciona con sus propios `conceptos_tenant`.

#### ¿Cómo funciona la "Magia" (El Flujo de Vida)?

1. **La Siembra (Seeding):** Cuando una nueva municipalidad se registra en tu sistema (o cuando tú actualizas las plantillas), tu código en Go lee la tabla `base_regimen_default`. Go dice: *"Ah, la ley dice que necesito el modelo X. Vamos a buscar cuál es el concepto espejo de ese modelo en esta municipalidad"*. Go encuentra el `concepto_tenant_id` correspondiente y **lo copia (lo siembra)** en la tabla `base_regimen_tenant`.
2. **La Personalización (Override):** Ahora la municipalidad entra a su panel de control. Ven su lista de conceptos afectos a la CTS. Como es *su* tabla (`base_regimen_tenant`), si ellos quieren quitar un concepto o agregar un bono loco que solo existe en su ciudad, lo hacen libremente. **No afectan a la ley global, ni afectan a otras municipalidades.**
3. **El Cálculo (Planilla):** Cuando llega fin de mes, tu motor de Go es increíblemente estúpido y rápido. **Ignora por completo la tabla Default**. Solo hace un query: `SELECT conceptos WHERE tenant_id = X AND beneficio = CTS`. Suma los montos y aplica la matemática.

---

### 3. Conceptos Involucrados en esta Solución

Para que domines este vocabulario, esto es lo que estamos aplicando:

* **Regla Dura (Hardcode):** Lo que hacías antes. `if concepto == "0121" { sumar() }`. Es malo porque requiere recompilar Go si la ley cambia.
* **Data-Driven Rule Engine (Motor de Reglas por Datos):** Lo que estás haciendo ahora. Go solo sabe hacer matemáticas (`(A+B)/12`). Qué cosas son `A` y `B` se lo dicta la base de datos (`base_regimen_tenant`).
* **Aislamiento Multitenant (Multi-tenant Isolation):** Garantizar que la data de un cliente no se filtre ni altere la de otro. Al tener una tabla exclusiva `base_regimen_tenant`, logras aislamiento perfecto.
* **Resolución en Tiempo de Configuración (No en Tiempo de Ejecución):** El análisis 1 menciona algo vital: *"El motor de cálculo no necesita lógica de 'si no hay override, usa el default'"*. Si tuvieras que decidir qué concepto usar mientras calculas la planilla de 1000 personas, el sistema sería lento. Al "sembrar" los datos antes, la planilla vuela porque la decisión ya está tomada en la base de datos.

### 4. Mi Recomendación Definitiva para la Implementación

Basado en todos los documentos que me has mostrado, mi recomendación como arquitecto es **Aprobar e implementar el modelo del ERD (Analisis 1)**. Es la solución más profesional, elegante y madura que podrías aplicar.

**El plan de ataque sería el siguiente:**

1. **Refactorizar la Base de Datos:** Crear las tablas `conceptos_calculados`, `base_regimen_default` y `base_regimen_tenant`. (Abandonar las columnas booleanas antiguas como `es_base_cts`).
2. **Crear el "Sembrador" en Go:** Una pequeña función en tus servicios que lea la tabla default y llene la tabla del tenant automáticamente cuando se crea un nuevo concepto o se inicializa un inquilino.
3. **Actualizar el Motor de Planilla:** Modificar `planilla_service.go` para que, al calcular beneficios, sume los montos cruzando la información con `base_regimen_tenant`.
4. **UI de Configuración:** Una pantalla sencilla en el Tenant donde el usuario vea "Variables de Cálculo" y pueda activar o desactivar *checkboxes* para sus conceptos.

Sé que suena a mucho trabajo, pero créeme, **este es el momento exacto en el ciclo de vida de tu software para hacerlo**. Estás quitándole los "rueditas" a la bicicleta para ponerle un motor de Fórmula 1.

---

# LA PROPUESTA
Para mantener el riesgo bajo y no romper la planilla regular actual, **esta primera fase implementará el motor de reglas de datos exclusivamente para el cálculo de Vacaciones (Liquidaciones)**. Una vez que esto funcione perfectamente, podrás migrar la CTS, Gratificaciones y demás beneficios a este mismo motor con un par de clics.

Aquí tienes el plan de implementación estructurado paso a paso.

---

## Plan de Implementación: Motor de Reglas Basado en Datos (Fase 1 - Estructura y Sembrador)

### Fase 1: Base de Datos (Esquema SQL de Siembra y Sobrescritura)

Vamos a crear el catálogo de fórmulas y separar la ley global (`default`) de la configuración local (`tenant`).

**Instrucciones para el Agente:**

1. Crear un archivo de migración SQL con las siguientes sentencias:

```sql
-- 1. Catálogo de Fórmulas y Beneficios
CREATE TABLE conceptos_calculados (
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(150) NOT NULL,
    tipo VARCHAR(50) NOT NULL,             -- Ej: 'BENEFICIO_SOCIAL'
    codigo_interno VARCHAR(50) UNIQUE NOT NULL, -- Ej: 'VAC_TRUNCAS', 'VAC_NO_GOZADAS'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Capa Global (La Ley / Súper Admin)
CREATE TABLE base_regimen_default (
    id SERIAL PRIMARY KEY,
    concepto_calculado_id INT NOT NULL REFERENCES conceptos_calculados(id) ON DELETE CASCADE,
    regimen_id INT NOT NULL REFERENCES regimenes_laborales(id) ON DELETE CASCADE,
    concepto_modelo_id INT NOT NULL REFERENCES conceptos_modelo(id) ON DELETE CASCADE,
    variable_calculo VARCHAR(50) NOT NULL, -- Ej: 'REMUNERACION_COMPUTABLE'
    UNIQUE(concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo)
);

-- 3. Capa Local (El Cliente / Municipalidad)
CREATE TABLE base_regimen_tenant (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    concepto_calculado_id INT NOT NULL REFERENCES conceptos_calculados(id) ON DELETE CASCADE,
    regimen_id INT NOT NULL REFERENCES regimenes_laborales(id) ON DELETE CASCADE,
    concepto_tenant_id INT NOT NULL REFERENCES conceptos_tenant(id) ON DELETE CASCADE,
    variable_calculo VARCHAR(50) NOT NULL,
    UNIQUE(tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo)
);

-- Índices de Rendimiento para el Motor de Go
CREATE INDEX idx_base_regimen_tenant_calc ON base_regimen_tenant(tenant_id, regimen_id, concepto_calculado_id);

```

### Fase 2: Modelos en Go

**Instrucciones para el Agente:**

1. En `internal/models/calculos.go` (o `core.go`), agregar las estructuras correspondientes:

```go
type ConceptoCalculado struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	Tipo          string `json:"tipo"`
	CodigoInterno string `json:"codigo_interno"`
}

type BaseRegimenDefault struct {
	ID                  int    `json:"id"`
	ConceptoCalculadoID int    `json:"concepto_calculado_id"`
	RegimenID           int    `json:"regimen_id"`
	ConceptoModeloID    int    `json:"concepto_modelo_id"`
	VariableCalculo     string `json:"variable_calculo"`
}

type BaseRegimenTenant struct {
	ID                  int    `json:"id"`
	TenantID            int    `json:"tenant_id"`
	ConceptoCalculadoID int    `json:"concepto_calculado_id"`
	RegimenID           int    `json:"regimen_id"`
	ConceptoTenantID    int    `json:"concepto_tenant_id"`
	VariableCalculo     string `json:"variable_calculo"`
}

```

### Fase 3: Capa de Repositorios (Lectura para el Cálculo)

Necesitamos el método que leerá la Capa Local para alimentar el motor matemático.

**Instrucciones para el Agente:**

1. Crear el archivo `internal/repository/base_regimen_repository.go` y la estructura `BaseRegimenRepository` con inyección de `*sql.DB`.
2. Implementar el método para que el motor de planillas sepa qué sumar:

```go
// ObtenerMontoVariable extrae la suma de los conceptos asignados a un puesto que pertenecen a una variable de cálculo específica (Ej: REMUNERACION_COMPUTABLE)
func (r *BaseRegimenRepository) ObtenerMontoVariable(tenantID, puestoID, regimenID int, codInternoCalculado, variableCalculo string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(pc.monto), 0.0)
		FROM puesto_conceptos pc
		INNER JOIN base_regimen_tenant brt ON pc.concepto_tenant_id = brt.concepto_tenant_id
		INNER JOIN conceptos_calculados cc ON brt.concepto_calculado_id = cc.id
		WHERE pc.puesto_id = $1 AND pc.activo = true 
		  AND brt.tenant_id = $2 AND brt.regimen_id = $3 
		  AND cc.codigo_interno = $4 AND brt.variable_calculo = $5
	`
	var monto float64
	err := r.db.QueryRow(query, puestoID, tenantID, regimenID, codInternoCalculado, variableCalculo).Scan(&monto)
	return monto, err
}

```

### Fase 4: El "Sembrador" en el Servicio de Conceptos

Aquí programamos la magia de sincronización: cuando se crea un tenant o se sincronizan los modelos, Go copia la "Ley" a la "Realidad" del Tenant.

**Instrucciones para el Agente:**

1. En `internal/services/concepto_modelo_service.go`, agregar el método `SembrarBaseRegimenTenant(tenantID int) error`:

```go
// SembrarBaseRegimenTenant copia la configuración global de cálculo hacia el Tenant, resolviendo los IDs espejo.
func (s *ConceptoModeloService) SembrarBaseRegimenTenant(tenantID int) error {
	query := `
		INSERT INTO base_regimen_tenant (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo)
		SELECT $1, brd.concepto_calculado_id, brd.regimen_id, ct.id, brd.variable_calculo
		FROM base_regimen_default brd
		INNER JOIN conceptos_tenant ct ON brd.concepto_modelo_id = ct.modelo_id
		WHERE ct.tenant_id = $1
		ON CONFLICT (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo) DO NOTHING;
	`
	_, err := s.Db.Exec(query, tenantID)
	return err
}

```

2. Modificar el método actual `SincronizarSaaS` (o equivalente) para que, inmediatamente después de clonar/actualizar los `conceptos_tenant`, ejecute `s.SembrarBaseRegimenTenant(tenant.ID)`.

---

### Siguientes pasos después de que el Agente termine

Cuando Antigravity CLI ejecute este plan, tendrás la infraestructura perfecta lista. En la base de datos se crearán las 3 tablas con sus relaciones.

**Lo que haremos nosotros manualmente por SQL luego:**
Insertaremos la semilla para las Vacaciones, algo así:

1. `INSERT INTO conceptos_calculados (nombre, tipo, codigo_interno) VALUES ('Vacaciones Truncas', 'BENEFICIO_SOCIAL', 'VAC_TRUNCAS');`
2. `INSERT INTO base_regimen_default ...` (Asignando el Sueldo Básico a las Vacaciones Truncas del Régimen 728).

Finalmente, ejecutaremos el botón Sincronizar en el panel Admin y ¡Pum! Todo se copiará a los tenants.
