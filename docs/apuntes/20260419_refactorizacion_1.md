# El Nuevo Plan de Implementación (Hoja de Ruta Refactorizada)
Para aplicar estas excelentes ideas sin perder lo que ya hemos avanzado, te propongo este nuevo orden de ataque:

## Fase A: Refactorización Arquitectónica (Limpieza)

1. Crear la carpeta `internal/services/`.

2. Mover la lógica pesada que teníamos en `planilla_repository`.go hacia un nuevo `planilla_service.go`.

3. Dejar el Repositorio solo con funciones de extracción (ej. `ObtenerContratosActivos`, `ObtenerConceptosPorPuestos`, `ObtenerAfectacionesAgrupadas`).

## Fase B: Optimización del Motor (Tu propuesta de Listas)

1. Construir la función que genera la "Lista de comparación de retenciones y aportaciones" cruzando la tabla `conceptos_afectaciones`.

2. Implementar la generación del diccionario de `bases_de_calculo` por `contrato_id` que diseñaste en tu archivo.

## Fase C: Concurrencia (Workers)

1. Envolver el motor en Goroutines.

2. (Opcional pero recomendado) Añadir un campo de "progreso" o "estado" a la tabla planillas para que el frontend pueda consultar mediante HTMX (hx-get con hx-trigger="every 2s") y mostrar una barra de progreso mientras los Workers hacen su trabajo.