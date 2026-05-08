# Plan de Implementación Definitivo
Con toda esta información, el plan exacto a seguir será:

## Fase 1: Actualización de Base de Datos (SQL)
Primero, debemos preparar el terreno en PostgreSQL.

Haremos un:

```SQL
ALTER TABLE conceptos_tenant ADD COLUMN requiere_monto BOOLEAN DEFAULT false;
```

Crearemos la nueva tabla `conceptos_modelo` con todas sus relaciones, incluyendo tu nueva columna.

```SQL
CREATE TABLE conceptos_modelo (
    id SERIAL PRIMARY KEY,
    regimen_id INTEGER NOT NULL REFERENCES regimenes_laborales(id),
    concepto_id INTEGER NOT NULL REFERENCES conceptos_maestros(id),
    nombre_personalizado VARCHAR(150) NOT NULL,
    frecuencia_meses VARCHAR(50) DEFAULT '1,2,3,4,5,6,7,8,9,10,11,12',
    clasificador_id INTEGER REFERENCES clasificadores_mef(id),
    es_extraordinario BOOLEAN DEFAULT false,
    requiere_monto BOOLEAN DEFAULT false, -- Tu nueva adición estrella
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## Fase 2: Módulo Administrativo (El CRUD del Super Admin)
Construiremos el panel para que el Super Admin llene el "molde".

-   **Modelo Go**: Crear la estructura `models.ConceptoModelo`.

-   **Repositorio** (`concepto_modelo_repository.go`): Funciones para crear, listar por régimen, editar y eliminar.

-   **Controlador** (`concepto_modelo_handler.go`): Para manejar las peticiones web y las vistas HTML del administrador.

-   **Vistas HTML**: Un nuevo ítem en el menú lateral y su respectiva interfaz con buscador y filtro por régimen laboral.

## Fase 3: Lógica de Clonación (El Motor)
En un servicio o en el mismo repositorio, crearemos una función llamada `ClonarModelosHaciaTenant(tenantID int) error`.

Esta función ejecutará un simple pero poderoso `INSERT INTO ... SELECT ...` a nivel de SQL. Tomará todo de `conceptos_modelo` y lo insertará en `conceptos_tenant` inyectando el nuevo `tenantID`.

Conectaremos esta función en admin_handler.go, justo después de crear el inquilino.

## Fase 4: Migración de Datos Base
Una vez creado el módulo, en lugar de depender de `config/plantillas_conceptos.go`, tú mismo desde el panel de Super Admin registrarás la Remuneración, el Aguinaldo, EsSalud y la Renta de 5ta para los regímenes 276, 728 y CAS. Una vez registrados, podremos eliminar de forma segura el archivo `plantillas_conceptos.go`.