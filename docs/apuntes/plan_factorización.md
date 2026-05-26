# Plan de refacrtorización
Estás experimentando exactamente lo que en la industria llamamos un **"Fat Handler" (Controlador Gordo)**, el cual viola directamente el Principio de Responsabilidad Única (SRP). Si el día de mañana deseas cambiar el tipo de letra de un reporte PDF o agregar una columna a un Excel, no deberías tener que tocar el archivo encargado de gestionar las peticiones HTTP del navegador.

## Evaluación Técnica y Diagnóstico
1. Consolidación de Modelos (`reporte.go` vs `reportes.go`)
   * **El Problema**: Al indicarle un prompt general, la CLI creó `reporte.go` desde cero ignorando la convención o existencia previa de `reportes.go`. Tener ambos archivos genera confusión en el equipo y rompe las convenciones de Go.

   * **La Solución**: En Go, la buena práctica dicta que los archivos de modelos y sus estructuras internas se nombren en singular (`reporte.go` y `type Reporte struct`). Por lo tanto, debemos eliminar por completo el archivo redundante `reportes.go` y asegurar que todas las importaciones apunten al modelo unificado `reporte.go`. Asegurar la miogración del contenido de `repores.go` antes de su eliminación.

2. Extracción de Lógica de Negocio hacia un `ReporteService`
   * **El Problema**: Actualmente, `reporte_handler.go` contiene todo el motor de formateo, las consultas a repositorios, las inicializaciones de `excelize.NewFile()`, los diseños de celdas y la escritura binaria. Es un archivo inmenso y acoplado.

   * **La Solución**: Introducir una capa intermedia: el `ReporteService`.

     * El Handler pasará a ser un componente "delgado": solo extraerá los parámetros de la URL (id, mes, dias), se los pasará al servicio, recibirá un buffer de bytes limpio (`*bytes.Buffer`) junto con el nombre sugerido del archivo, y lo escupirá al navegador.

     * El **Servicio** se encargará de orquestar: llamará a los repositorios correspondientes, procesará las reglas de negocio (como calcular quiénes cumplen años) y construirá los documentos.

## Plan de Refactorización Quirúrgica para Antigravity CLI

### Fase 1: Limpieza de Modelos Duplicados
1. Examinar `internal/models/`. Eliminar definitivamente el archivo `reportes.go`. Antes de eliminarlo migrar su contenido correctamente.

2. Mantener únicamente `internal/models/reporte.go` con el struct `type Reporte struct`.

3. Escanear el proyecto (`go check` o búsquedas de texto) para asegurar que ninguna ruta o handler haya quedado apuntando al archivo eliminado.

### Fase 2: Creación de la Capa de Servicio (`internal/services/reporte_service.go`)
1. Crear el archivo `internal/services/reporte_service.go`.

2. Definir el struct `ReporteService` inyectándole todos los repositorios que el handler usaba anteriormente:

```Go
package services

import (
    "bytes"
    "planilla-rgm/internal/repository"
    "github.com/xuri/excelize/v2"
    "github.com/jung-kurt/gofpdf"
)

type ReporteService struct {
    TrabajadorRepo     *repository.TrabajadorRepository
    OrganigramaRepo    *repository.OrganigramaRepository
    PuestoRepo         *repository.PuestoRepository
    ConceptoTenantRepo *repository.ConceptoTenantRepository
    ContratoRepo       *repository.ContratoRepository
}
```

3. Mover los pesados bloques switch id desde el handler hacia dos métodos limpios en el servicio:

   * `GenerarPDF(tenantID int, id string, params map[string]string) (*bytes.Buffer, string, error)`

   * `GenerarExcel(tenantID int, id string, params map[string]string) (*bytes.Buffer, string, error)`
   *(__Nota para el agente__: En lugar de escribir directamente en `w (http.ResponseWriter)`, el servicio creará un `var buf bytes.Buffer` y usará `f.Write(&buf)` o `pdf.Output(&buf)` para retornar los bytes puros en memoria)*.

### Fase 3: Adelgazamiento Extremo del Handler (`reporte_handler.go`)
1. Modificar `internal/handlers/reporte_handler.go`. Quitarle todas las dependencias directas de repositorios e inyectarle únicamente el nuevo `Service *services.ReporteService`.

2. Reducir los métodos `ExportarPDF` y `DescargarExcel` a su mínima expresión. Por ejemplo, el método de Excel debería quedar tan limpio como esto:

```Go
func (h *ReporteHandler) DescargarExcel(w http.ResponseWriter, r *http.Request) {
    tenantID := obtenerTenantID(r)
    id := r.URL.Query().Get("id")
    
    // Capturamos cualquier parámetro dinámico opcional (mes, dias, etc.)
    params := map[string]string{
        "mes":  r.URL.Query().Get("mes"),
        "dias": r.URL.Query().Get("dias"),
    }

    // Delegamos TODA la carga pesada al servicio institucional
    buffer, nombreArchivo, err := h.Service.GenerarExcel(tenantID, id, params)
    if err != nil {
		http.Error(w, "Error al generar el reporte: "+err.Error(), http.StatusInternalServerError)
		return
	}

    // El Handler solo se dedica a su verdadera función: responder las cabeceras HTTP
    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nombreArchivo))
    
    // Transmitimos los bytes directamente al navegador del usuario
    buffer.WriteTo(w)
}
```

### Fase 4: Actualización de Rutas (`internal/routes/routes.go`)
1. En tu archivo de rutas, inicializar el servicio pasándole los repositorios:
   `reporteService := services.NewReporteService(trabajadorRepo, organigramaRepo, puestoRepo, conceptoRepo, contratoRepo)`

2. Inyectar este servicio en la inicialización del handler de reportes antes de registrar los endpoints.

3. Ejecutar las pruebas automatizadas del proyecto (`go test ./...`) para certificar el éxito total de la refactorización.

**¿Qué ganamos con este cambio arquitectónico?**
**Mantenibilidad de Clase A**: Si decides cambiar los colores corporativos de los encabezados de las tablas de Excel para que combinen con la municipalidad, sabrás con precisión milimétrica que debes ir a `reporte_service.go`, sin tocar nada del tráfico de red web.

**Facilidad de Testeo**: Ahora podrás crear pruebas unitarias (`_test.go`) en Go para verificar que las celdas y los cálculos matemáticos de los reportes se armen correctamente sin necesidad de simular servidores web ni peticiones HTTP artificiales.