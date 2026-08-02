# Sobre el Anexo 1A de la planilla

## El Anexo 1A Resumen por conceptos de planilla
Está referido a un informe donde se pueden observar los conceptos de la planilla en el periodo de trabajo, organizados por tipo de concepto (INGRESO, RETENCIÓN Y APORTE), totalizados a nivel de conceptos y de tipo de concepto, que deben sumar el "monto total de la planilla" (Recuerda que solo debemos sumar INGRESO y APORTE).

Los ajustes de redondeo (ONP, renta de 4ta y renta de 5ta) deben sumarse directamente a los totales de los conceptos tenant: "SNP DL 19990 - ONP" (0607), "Renta de Cuarta Categoría - Retenciones" (S101) y "Renta de Quinta Categoría - Retenciones" (0605), de tal manera que el total muestre el "Costo Total de la Planilla".

### Diagrama de relaciones entre tablas

```mermaid
erDiagram
	planillas ||--o{ planilla_detalles : has
	planillas {
		serial id PK		
	}
	planilla_detalles ||--o{ planilla_conceptos : has
	planilla_detalles {
		serial id PK
		integer planilla_id FK
		integer contrato_id FK
	}
	planilla_conceptos {
		serial id PK
		integer planilla_detalle_id FK
		integer concepto_tenant_id FK
		character(20) tipo_concepto
		numeric(10-2) monto
		integer maestro_id
		character(10) codigo_sunat
		character(150) nombre_en_boleta
	}
	conceptos_tenant ||--o{ planilla_conceptos : has
	conceptos_tenant {
		serial id PK
		integer clasificador_id FK
	}
	contratos ||--o{ planilla_detalles : has
	contratos {
		serial id PK
		integer trabajador_id FK
		integer puesto_id FK
	}
	trabajadores ||--o{ contratos : has
	trabajadores {
		serial id PK
	}
	puestos ||--o{ contratos : has
	puestos {
		serial id PK
		integer meta_id FK
	}
	metas_presupuestales ||--o{ puestos : has
	metas_presupuestales {
		serial id PK
	}
	clasificador_mef ||--o{ conceptos_tenant : has
	clasificador_mef ||--o{ clasificador_mef : has
	clasificador_mef ||--o{ conceptos_modelo : has
	clasificador_mef {
		serial id PK
		integer parent_id FK
	}
	conceptos_modelo {
		serial id PK
		integer clasificador_id FK
	}
```
