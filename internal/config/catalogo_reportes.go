package config

import "planilla-rgm/internal/models"

var ListaReportes = []models.Reporte{
	{ID: "trab_padron", Modulo: "TRABAJADORES", Nombre: "👥 Padrón General de Personal", Descripcion: "Listado completo de trabajadores activos con datos de contacto, régimen pensionario y legajo."},
	{ID: "trab_cumple", Modulo: "TRABAJADORES", Nombre: "🎂 Cumpleaños del Mes", Descripcion: "Personal que celebra su onomástico en el mes seleccionado. Ideal para bienestar social."},
	{ID: "org_directorio", Modulo: "ORGANIGRAMA", Nombre: "🏢 Directorio de Dependencias", Descripcion: "Estructura orgánica completa basada en la ordenanza municipal vigente y códigos MEF."},
	{ID: "puesto_resumen", Modulo: "PUESTOS", Nombre: "📊 Ocupabilidad de Plazas (CAP/PAP)", Descripcion: "Cuadro estadístico de puestos en estado VACANTE u OCUPADO por unidad orgánica."},
	{ID: "puesto_pap", Modulo: "PUESTOS", Nombre: "💰 Presupuesto Analítico (PAP) Resumido", Descripcion: "Resumen de costos mensuales presupuestados asignados por plaza."},
	{ID: "concepto_maestro", Modulo: "CONCEPTOS", Nombre: "⚙️ Catálogo Local de Conceptos", Descripcion: "Lista de ingresos y aportes configurados con sus afectaciones de ley y clasificadores MEF."},
	{ID: "contrato_vence", Modulo: "CONTRATOS", Nombre: "⏳ Alertas de Vencimiento", Descripcion: "Contratos de personal a plazo fijo o transitorios CAS próximos a culminar en el rango de días especificado."},
}
