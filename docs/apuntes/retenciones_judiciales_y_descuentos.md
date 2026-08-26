# Retenciones judiciales

## El Problema
Las retenciones judiciales son conceptos que se calculan en base a un porcentaje o monto fijo de los ingresos de un trabajador, debidamente ordenados por un juez.
El trabajador tiene diferentes conceptos en su planilla, que pueden ser permanentes (todos los meses), extraordinarios (autorizados por norma legal), periódicos (Aguinaldos, Gratificaciones, CTS) y ocasionales (CTS, Vacaciones, y otros). Es necesario definir a qué conceptos aplica la retención judicial.
Debido a que es necesario una trazabilidad los conceptos deben especificarse uno por uno, según la estructura de costos (`puesto_conceptos`) del puesto activo asignado al trabajador en la tabla `contratos` (también activo).
Lo cierto es que en las municipalidades los trabajadores autorizan varios tipos de descuentos:
- Retenciones judiciales por alimentos.
- Retenciones judiciales por deudas.
- Descuentos por sindicatos.
- Descuentos por prestamos (de entidades financieras).
- Descuentos por contribuciones permitidas por ley.
- Otros descuentos.
Por eso, la solución debería permitir configuraciones flexibles para cada trabajador y tipo de descuento autorizado.

## Solución Propuesta
La propuesta de solución es más bien "generalista" que puede aplicarse a las retenciones judiciales o a otro tipo de retenciones, consiste en:

Crear dos tablas adicionales:
- *La tabla `descuentos` sea la que almacene los datos de los descuentos, ya sean por orden judicial, por organización sindical, por contribuciones permitidas por ley, etc.
- *La tabla `descuento_conceptos` sea la que almacene los datos de los conceptos a los que se les aplica el descuento.

La tabla `descuentos` podría ser de la siguiente manera:

```sql
CREATE TABLE descuentos (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    trabajador_id INT NOT NULL,
    tipo_descuento CHARACTER VARYING(50) DEFAULT 'JUDICIAL'::character varying,
    documento_ordenador CHARACTER VARYING(50) DEFAULT 'EXPEDIENTE'::character varying,
    detalle_documento CHARACTER VARYING(255),
    detalle_descuento CHARACTER VARYING(255) NOT NULL,
    entidad_financiera_id INT NOT NULL,
    monto_fijo DECIMAL(10,2) NOT NULL,
    porcentaje DECIMAL(5,2) NOT NULL,
    activo BOOLEAN DEFAULT true NOT NULL,
    motivo_baja CHARACTER VARYING(255) NOT NULL,
    inicio_vigencia TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    fin_vigencia TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    beneficiario_nombre CHARACTER VARYING(150) NOT NULL,
    beneficiario_cuenta CHARACTER VARYING(25) NOT NULL,
    beneficiario_cci CHARACTER VARYING(25) NOT NULL,
    beneficiario_numero_documento CHARACTER VARYING(15) NOT NULL,
    beneficiario_tipo_documento CHARACTER VARYING(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (trabajador_id) REFERENCES trabajadores(id),
    FOREIGN KEY (entidad_financiera_id) REFERENCES entidades_financieras(id),
    CONSTRAINT chk_monto_porcentaje CHECK (monto_fijo > 0 OR porcentaje > 0),
    CONSTRAINT chk_vigencia CHECK (fin_vigencia >= inicio_vigencia),
    CONSTRAINT chk_tipo_descuento CHECK ((tipo_descuento)::text = ANY ((ARRAY['JUDICIAL'::character varying, 'SINDICAL'::character varying, 'PRESTAMO'::character varying, 'CONVENIO'::character varying, 'OTROS'::character varying])::text[])),
    CONSTRAINT chk_documento_ordenador CHECK ((documento_ordenador)::text = ANY ((ARRAY['SENTENCIA'::character varying, 'RESOLUCION'::character varying, 'CONTRATO'::character varying, 'OTRO'::character varying])::text[])),
    CONSTRAINT chk_tipo_documento CHECK ((beneficiario_tipo_documento)::text = ANY ((ARRAY['DNI'::character varying, 'RUC'::character varying, 'PASAPORTE'::character varying, 'CARNET_EXTRANJERIA'::character varying])::text[]))
);
```

### Explicación
* **tenant_id**: ID del tenant al que pertenece el descuento.
* **trabajador_id**: ID del trabajador al que se le aplica el descuento.
* **tipo_descuento**: Tipo de descuento.
* **documento_ordenador**: Tipo de documento que ordena el descuento.
* **detalle_documento**: Número del documento que ordena el descuento.
* **detalle_descuento**: Descripción del descuento.
* **entidad_financiera_id**: ID de la entidad financiera ordenadora del descuento.
* **monto_fijo**: Monto fijo a descontar.
* **porcentaje**: Porcentaje a descontar.
* **activo**: Indica si el descuento está activo.
* **motivo_baja**: Motivo de la baja del descuento.
* **inicio_vigencia**: Fecha de inicio de vigencia del descuento.
* **fin_vigencia**: Fecha de fin de vigencia del descuento.
* **beneficiario_nombre**: Nombre del beneficiario del descuento.
* **beneficiario_cuenta**: Cuenta del beneficiario del descuento.
* **beneficiario_cci**: Cuenta Interbancaria del beneficiario del descuento.
* **beneficiario_numero_documento**: Número de documento del beneficiario del descuento.
* **beneficiario_tipo_documento**: Tipo de documento del beneficiario del descuento.
* **created_at**: Fecha de creación del descuento.
* **updated_at**: Fecha de actualización del descuento.

La tabla `descuento_conceptos` podría ser de la siguiente manera:

```sql
CREATE TABLE descuento_conceptos (
    id INT AUTO_INCREMENT PRIMARY KEY,
    descuento_id INT NOT NULL,
    concepto_tenant_id INT NOT NULL,
    porcentaje DECIMAL(5,2) NOT NULL,
    monto_fijo DECIMAL(10,2) NOT NULL,
    activo BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (descuento_id) REFERENCES descuentos(id),
    FOREIGN KEY (concepto_tenant_id) REFERENCES conceptos_tenant(id),
    CONSTRAINT chk_monto_porcentaje CHECK (monto_fijo > 0 OR porcentaje > 0)
);
```

### Explicación
* **descuento_id**: ID del descuento.
* **concepto_tenant_id**: ID del concepto al que se le aplica el descuento.
* **porcentaje**: Porcentaje a descontar.
* **monto_fijo**: Monto fijo a descontar.
* **activo**: Indica si el concepto está activo.
* **created_at**: Fecha de creación del concepto.
* **updated_at**: Fecha de actualización del concepto.

---
## Consideraciones del Frontend
En el frontend se debe tener en cuenta lo siguiente:
- Una tabla maestra donde se muestren los decuentos por cada trabajador (puede existir más de un descuento por trabajador).
- Un modal para crear o editar un descuento, con campos para:
    - Tipo de descuento.
    - Documento ordenador.
    - Detalle del documento.
    - Detalle del descuento.
    - Entidad financiera (implementar TomSelect).
    - Monto fijo.
    - Porcentaje.
    - Activo.
    - Motivo de baja.
    - Inicio de vigencia.
    - Fin de vigencia.
    - Beneficiario nombre.
    - Beneficiario cuenta.
    - Beneficiario CCI.
    - Beneficiario número de documento.
    - Beneficiario tipo de documento.
- Una vista de detalle para ver un descuento en específico donde se muestra una tabla con los conceptos afectados.
- En la vista de detalle se debe mostrar una tabla con los conceptos que están siendo afectados por el descuento y el porcentaje o monto fijo que se le está aplicando a cada concepto.
- Un modal para agregar los conceptos afectados al descuento, con campos para:
    - Concepto tenant (implementar TomSelect).
    - Porcentaje.
    - Monto fijo.
    - Activo.
    - Motivo de baja.
    - Inicio de vigencia.
    - Fin de vigencia.

---
## Consideraciones del Backend
En el backend se debe tener en cuenta lo siguiente:
- Un servicio para gestionar los descuentos manejando la lógica de:
    - Crear o editar un descuento (descuentos y descuento_conceptos).
    - Eliminar un descuento.
    - Obtener los descuentos por trabajador a petición del proceso de la planilla mensual.
    - Validar la lógica de los descuentos a petición del proceso de la planilla mensual.

---
## Consideraciones de la Base de Datos
En la base de datos se debe tener en una tabla `entidades_financieras`:

```sql
CREATE TABLE entidades_financieras (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    nombre CHARACTER VARYING(255) NOT NULL,
    activo BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT chk_activo CHECK ((activo)::text = ANY ((ARRAY['true'::character varying, 'false'::character varying])::text[]))
);
```

Se utiliza como base la lista de entidades financieras de la SUNAT:

codigo,entidad_financiera
002,"BANCO DE CRÉDITO DEL PERÚ"
003,"INTERBANK"
007,"CITIBANK DEL PERÚ"
009,"SCOTIABANK PERÚ"
011,"BBVA BANCO CONTINENTAL"
018,"BANCO DE LA NACIÓN"
020,"BANCO FALABELLA"
023,"BANCO DE COMERCIO"
035,"BANCO PICHINCHA"
038,"BANCO INTERAMERICANO DE FINANZAS"
043,"CREDISCOTIA FINANCIERA"
053,"BANCO GNB"
056,"SANTANDER"
057,"BANCO AZTECA"
058,"BANCO CENCOSUD"
059,"BANCO RIPLEY"
060,"ICBC PERÚ BANK"
070,"MIBANCO"
200,"FINANC. CREDINKA"
202,"FINANC. PROEMPRESA"
204,"FINANC. CONFIANZA"
206,"CREDIRAIZ"
208,"COMPARTAMOS FINANCIERA"
210,"FINANCIERA QAPAQ"
212,"FINANCIERA TFC S A"
214,"FINANCIERA EFECTIVA"
216,"AMERIKA FINANCIERA"
218,"FINANCIERA OH!"
800,"CAJA METROPOLITANA DE LIMA"
802,"CMAC TRUJILLO"
803,"CMAC AREQUIPA"
805,"CMAC SULLANA"
806,"CMAC CUSCO"
808,"CMAC HUANCAYO"
813,"CMAC TACNA"
820,"CMAC DEL SANTA"
822,"CMAC ICA"
824,"CMAC PIURA"
826,"CMAC MAYNAS"
828,"CMAC PAITA"
900,"CRAC SIPAN"
902,"CRAC DEL CENTRO"
904,"CRAC INCASUR"
906,"CRAC PRYMERA"
908,"CRAC LOS ANDES"
