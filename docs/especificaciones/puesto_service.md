# Especificaciones del Servicio: PuestoService

Este documento detalla la lógica de negocio central de la creación de puestos (plazas) y su inicialización de conceptos base, encapsulada en el servicio `PuestoService`.

---

## 1. Artefactos Involucrados

Los principales archivos y artefactos de código fuente que componen este flujo son:

*   **Manejador HTTP (Handler):** `internal/handlers/puesto_handler.go` (`PuestoHandler.Crear`).
*   **Capa de Servicios:** `internal/services/puesto_service.go` (`PuestoService`).
*   **Repositorio (Persistencia):** `internal/repository/puesto_repository.go` (`PuestoRepository`).
*   **Modelo de Datos:** `internal/models/core.go` (Struct `Puesto`).

---

## 2. Flujo de Acción y Funciones

El punto de entrada de este flujo ocurre cuando un usuario envía el formulario para crear una nueva plaza, el cual es interceptado por la función `Crear` del `PuestoHandler`. El Handler recolecta la información y delega la responsabilidad al servicio mediante `CrearPuestoConPlantilla`.

### `services.PuestoService`

*   **Función:** `CrearPuestoConPlantilla(nuevoPuesto *models.Puesto) error`
*   **Parámetros:** Un puntero a la estructura `models.Puesto` que contiene la información básica de la plaza (meta, fuente, régimen, sueldo, etc.).
*   **Flujo de Ejecución:**
    1.  **Persistencia Inicial:** Llama a `s.Repo.Crear(nuevoPuesto)` para registrar la plaza en la base de datos y obtener su nuevo `ID` generado. Si ocurre un error, se aborta la operación y se retorna el error.
    2.  **Carga de Plantilla (MAGIA SAAS):** Invoca a `s.Repo.ObtenerConceptosModeloPorRegimen` utilizando el `TenantID` y el `RegimenID` del puesto para obtener un arreglo con los IDs de los conceptos que aplican por defecto para dicho régimen. Esta consulta es "inteligente" ya que excluye automáticamente los conceptos previsionales (como ONP o AFP) aplicando una regla de negocio. Si esta operación falla, se registra un log del error, pero la creación del puesto continúa (se devuelve `nil`), permitiendo crear el puesto sin conceptos iniciales y sin interrumpir el proceso.
    3.  **Asignación Masiva:** Una vez obtenidos los IDs locales, llama a `s.Repo.AsignarConceptosAPuesto` para realizar una inserción masiva (Bulk Insert) y vincular de forma automática los conceptos obtenidos con la nueva plaza creada.

---

## 3. Funciones del Repositorio Invocadas

Las operaciones en la base de datos realizadas desde el servicio se delegan al `PuestoRepository`:

1.  `Crear(p *models.Puesto)`
    *   **Acción:** Realiza un `INSERT INTO puestos (...) VALUES (...) RETURNING id`. Registra el estado por defecto como 'VACANTE'.
    *   **Propósito:** Crear el registro base de la plaza y retornar su ID.

2.  `ObtenerConceptosModeloPorRegimen(tenantID int, regimenID int)`
    *   **Acción:** Realiza un `INNER JOIN` complejo entre las tablas `conceptos_modelo`, `regimen_concepto_modelo`, `conceptos_tenant` y `conceptos_maestros`.
    *   **Propósito:** Encontrar todos los conceptos locales (del Tenant) que corresponden al "modelo" o "plantilla" asociado al régimen laboral especificado. Aplica una cláusula `NOT IN ('0601', '0606', '0607', '0608')` sobre los códigos SUNAT para excluir intencionalmente las aportaciones previsionales. Retorna un arreglo de `int` (`[]int`) con los IDs de la tabla `conceptos_tenant`.

3.  `AsignarConceptosAPuesto(puestoID int, conceptoTenantIDs []int, sueldoBase float64)`
    *   **Acción:** Inicia una transacción SQL (`tx.Begin()`) y prepara una consulta `INSERT INTO puesto_conceptos (...) ON CONFLICT DO NOTHING`. Itera sobre el arreglo de IDs pasados como argumento y ejecuta el insert para cada uno.
    *   **Propósito:** Crear las configuraciones de costos iniciales para la plaza en un solo lote (Bulk Insert), asegurando que se guarden todos los registros de forma transaccional. La cláusula `ON CONFLICT` protege la base de datos contra inserciones duplicadas (idempotencia).

---

## 4. Aspectos Importantes del Servicio (MAGIA SAAS)

El aspecto más destacado del flujo en `PuestoService` es su enfoque automatizado SaaS para la **inicialización de recursos**. Cuando un inquilino crea una nueva plaza de trabajo, no necesita recordar ni configurar manualmente y desde cero todos los ingresos, retenciones o aportaciones del estado que aplican por ley al puesto creado.

Al delegar la responsabilidad a la **Plantilla Base (Modelo)** definida para cada régimen laboral (ej. D.L. 276, CAS), el sistema autoconfigura la estructura de pago inmediatamente. 

La exclusión intencional de los conceptos previsionales (AFP, ONP) durante esta asignación inicial automatizada es una decisión de diseño y de negocio clave. Se asume que la elección de afiliación a una administradora de fondos de pensión es de carácter personal e individual, y por tanto, su configuración debe darse en una etapa posterior (como al firmar un contrato vinculando un trabajador a la plaza).
