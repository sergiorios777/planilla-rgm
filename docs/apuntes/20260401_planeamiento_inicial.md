# PROYECTO: PLANILLA-RGM

## 1. Definición de Requisitos del Proyecto
Basado en tu descripción, aquí están los módulos clave y lo que debe contener cada uno:

### Panel del Administrador SaaS (Super Admin)
Este panel es global y no tiene `tenant_id` (o su tenant es 0/null).

* **Gestión de Inquilinos (Tenants):** Registro, suspensión o eliminación de las entidades (clientes) que usarán el software.

* **Gestión de Catálogos Universales (MEF):** Mantenimiento de los clasificadores presupuestales (ingresos y gastos).

* **Gestión de Parámetros Globales:** Configuración de valores anuales que afectan a todos, como la UIT (Unidad Impositiva Tributaria), RMV (Remuneración Mínima Vital), tasas de impuestos, etc.

* **Conceptos Maestros de Planilla:** Catálogo base de conceptos de ingresos (sueldo base, bonificaciones) y retenciones (AFP, ONP, quinta categoría).

### Panel del Inquilino (Tenant)
Toda consulta a la base de datos en este panel debe incluir WHERE tenant_id = ? por seguridad.

* **Administración Local:** Gestión de usuarios (operadores de planilla, recursos humanos) y roles dentro de esa entidad específica.

* **Módulo de Trabajadores (Legajo):**

  * Alta y baja de personal.

  * Gestión de contratos y regímenes laborales (ej. 276, 728, CAS, Ley del Servicio Civil).

  * Asignación de conceptos de ingresos y retenciones específicos por trabajador.

* **Módulo de Configuración Presupuestal:**

  * Creación de metas presupuestarias y fuentes de financiamiento.

  * Vinculación de los conceptos de planilla a los clasificadores presupuestarios del MEF habilitados para la entidad.

* **Módulo de Planillas:**

  * Cálculo y generación de la planilla mensual.

  * Cierre de planilla.

* **Módulo de Informes:**

  * Estimación anual del costo de planilla por metas, clasificadores y fuentes de financiamiento.

  * Costo anual detallado por trabajador o puesto.

## 2. Estructura de Carpetas Propuesta
Para mantener el código ordenado, escalable y separar la lógica de negocio de la interfaz visual, te sugiero esta estructura de directorios:

```plaintext
planilla-mef/
├── cmd/
│   └── server/
│       └── main.go          # Punto de entrada de la aplicación. Configura rutas y arranca el servidor.
├── internal/                # Código privado de tu aplicación (no exportable)
│   ├── config/              # Carga de variables de entorno (conexión a DB, puertos)
│   ├── models/              # Estructuras (Structs) de Go que representan tus tablas (User, Tenant, Worker...)
│   ├── repository/          # Lógica de base de datos (SQL). Aquí va el filtro estricto del tenant_id.
│   ├── service/             # Lógica de negocio (Ej. Los cálculos matemáticos de la planilla).
│   └── handlers/            # Controladores HTTP. Reciben peticiones HTMX y devuelven plantillas.
├── pkg/                     # Código público o utilidades (Ej. formateo de moneda, validación de DNI).
├── ui/                      # Todo lo relacionado al Frontend
│   ├── static/              # Archivos estáticos
│   │   ├── css/             # Tu archivo pico.css modificado o estilos personalizados
│   │   ├── js/              # Scripts adicionales (si necesitas algo más allá de Alpine.js)
│   │   └── img/             # Logos e imágenes
│   └── templates/           # Plantillas HTML procesadas por Go
│       ├── layouts/         # Plantillas base (header, footer, menú de navegación)
│       ├── admin/           # Vistas del Super Administrador SaaS
│       ├── tenant/          # Vistas de los Inquilinos (Trabajadores, Planillas, Informes)
│       └── components/      # Fragmentos HTML pequeños para devolver a HTMX (ej. filas de tablas, modales)
├── go.mod                   # Archivo de dependencias de Go
└── go.sum                   # Sumas de verificación de dependencias
```

## Instrucciones de Implementación (Estructura)
1. Abre tu terminal en la carpeta principal de tu proyecto.

2. Crea los directorios principales ejecutando:
   `mkdir -p cmd/server internal/{config,models,repository,service,handlers} pkg ui/{static/{css,js,img},templates/{layouts,admin,tenant,components}}`

3. Mueve tu archivo `main.go` actual a la carpeta `cmd/server/main.go`.

Esta estructura te asegurará que cuando tu aplicación crezca para manejar 100 inquilinos y cálculos complejos, sepas exactamente dónde encontrar un error de base de datos (`repository`), un error de cálculo (`service`) o un problema visual (`ui/templates`).