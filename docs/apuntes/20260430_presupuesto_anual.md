# Plan de Implementación (Versión ERP)
Dado que este módulo es complejo, lo dividiremos en 4 grandes fases. (Hoy podemos empezar con la Fase 1).

## FASE 1: Estructura de Datos y Dietas (Regidores)
1. **Modificar BD (`puestos`):** Añadir la columna `es_dietario` `BOOLEAN DEFAULT FALSE`.  
2. **Tablas de Presupuesto:** Crear `pap_versiones` y `pap_detalles` en PostgreSQL para guardar los reportes.  
3. **Catálogos:** Insertar el concepto "S102 - Dietas" en `conceptos_maestros`.  
4. **Ajuste UI y Filtro:** Actualizar la vista de puestos para incluir el check de "Es Puesto de Dieta", y actualizar el `planilla_repository.go` para excluir a los dietarios (`es_dietario = false`) de las planillas mensuales normales.  

## FASE 2: El Simulador de Costos
1. **Adaptar `calculos.go`:** Crear una función (ej. `SimularCostoPuesto`) que reciba un Puesto genérico. Se le inyectarán los parámetros globales (UIT, RMV, etc.) y ejecutará la lógica matemática asumiendo 0 faltas y un mes estándar.
2. **Extracción de Costo Real:** Esta función devolverá una estructura simple con dos totales: `Total Ingresos Brutos` y `Total Aportes Patronales` (los que impactan a la Municipalidad).

## FASE 3: Algoritmos de Proyección (El Motor PAP)
1. **Desarrollo del Servicio `pap_service.go`:**
2. **Método `GenerarPIA`:** Recibe el "Mes Base", obtiene todos los puestos activos, pasa cada uno por el simulador y distribuye los montos en los 12 meses iterando sobre `frecuencia_meses`.  
3. **Método `GenerarProyeccionVigente`:** Hace el `SELECT` a las planillas históricas para los `meses` cerrados , y usa el `SimularCostoPuesto` para llenar los meses vacíos.  
4. **Guardado:** Ambos métodos terminan haciendo un "Bulk Insert" masivo en la tabla `pap_detalles` vinculada a una nueva versión.  

## FASE 4: Interfaz de Usuario y Matriz Pivot
1. **Vista de Control:** Una pantalla donde el usuario decide qué tipo de proyección hacer (PIA o Año Vigente) y selecciona el Mes Base.  
2. **Renderizado Matricial:** Una tabla potente (HTML) que lea la versión guardada en `pap_detalles` y la agrupe dinámicamente: Meta -> Fuente -> Clasificador -> Mes 1 al 12 -> Total.