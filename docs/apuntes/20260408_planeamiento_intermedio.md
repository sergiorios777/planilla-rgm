# 🗺️ NUEVO PLAN DE IMPLEMENTACIÓN ACTUALIZADO
Con esta nueva visión, el proyecto evoluciona de ser una simple calculadora a un verdadero Sistema de Planificación y Gestión de RRHH. Aquí está la hoja de ruta actualizada:

## FASE 1: Estructura Organizacional y Presupuestal Base
1.1 Metas Presupuestales: Creación de la tabla `metas` vinculada al `tenant_id` y al `anio`.
1.2 Puestos (El nuevo corazón): Creación de la tabla `puestos` con su sueldo proyectado (independiente de si hay contrato o no).
1.3 Refactorización de Contratos: Modificar la tabla `contratos` para que el `Trabajador` ocupe un `Puesto`, heredando automáticamente su meta y régimen.

## FASE 2: Matriz de Conceptos, Clasificadores y Frecuencias
2.1 Frecuencias de Pago: Configurar la tabla o lógica que define en qué meses se paga cada concepto (Ej. Aguinaldo en julio/diciembre, Escolaridad en enero).
2.2 Vinculación MEF-PLAME (Tenant): Interfaz para que el inquilino relacione los conceptos del Súper Admin con sus clasificadores presupuestarios. Dejaremos sugerencias predefinidas pero editables.
2.3 Asignación Puesto-Conceptos: Asociar a cada puesto qué conceptos remunerativos le corresponden (Sueldo, Bonos, Retenciones) para costear su plaza.

## FASE 3: El Motor de Planilla (Procesamiento)
3.1 Planilla Mensual: Interfaz para generar la planilla de un mes específico.
3.2 Motor de Cálculo en Go: Implementación de las reglas matemáticas (`tipo_calculo`) para retenciones y aportaciones.
3.3 Cierre y Aprobación: Congelamiento de datos para emitir boletas de pago.

## FASE 4: Proyección y Reportes (El Objetivo Final)
4.1 Presupuesto Anualizado (PIA/PIM): El gran reporte que suma el costo de todos los `puestos` (vacantes y ocupados) cruzado por `Metas`, `Fuentes/Rubros` y `Clasificadores` para todo el año.