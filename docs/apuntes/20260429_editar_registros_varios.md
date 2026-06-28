# 📋 Plan de Batalla: Gestión y Edición de Módulos
## 1. El "Bug" de Puestos (Reparación Rápida)
* **Problema**: El botón cancelar no funciona.
* **Causa Probable**: Al igual que nos pasó con los conceptos locales, probablemente falta la ruta `/tenant/puestos/formulario-crear` o el botón no está apuntando al contenedor correcto (`#contenedor-formulario`).
* **Acción**: Implementar el mismo patrón de "Formulario Limpio" que usamos ayer.

## 2. Gestión de Contratos Laborales (Edición por Formulario)
* **Estrategia**: Dado que un contrato tiene relaciones con Trabajadores, Puestos y Regímenes, usaremos el patrón de Componente de Formulario (no edición en línea).
* **Flujo**:

    1. El botón ✏️ en la tabla de contratos llama a `/tenant/contratos/editar-ui?id=....`.
    2. El `EditarUI` carga el contrato y los desplegables de trabajadores y puestos.
    3. El `Update` guarda los cambios y dispara un `HX-Trigger: recargarTablaContratos`.
    4. El formulario vuelve a su estado de "Nuevo Contrato".

## 3. Metas Presupuestales y Asistencias (Edición Flexible)
* **Propuesta**: Para mantener la coherencia visual de todo el sistema, recomiendo usar Formulario de Edición (el mismo patrón anterior) para ambos.
* **Razón**: Aunque sean pocos campos, el usuario ya se acostumbró a que "Editar" cambia el formulario de la izquierda/arriba. Esto evita que el usuario tenga que aprender dos formas distintas de editar dentro de la misma aplicación.
* **Detalle en Asistencias**: Incluiremos la edición de `tipo`, `cantidad` y `fecha_ocurrencia`.