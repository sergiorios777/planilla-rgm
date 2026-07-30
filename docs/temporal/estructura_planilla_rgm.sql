--
-- PostgreSQL database dump
--

\restrict DlIhE8ov2sKGqW0fdEPEhxnbXD3P5JnesY0TcXYbn38arWaMjVMaCNtiWJbXNBi

-- Dumped from database version 18.4
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: admin_tareas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_tareas (
    id integer NOT NULL,
    titulo character varying(150) NOT NULL,
    descripcion text,
    recurrencia character varying(20) NOT NULL,
    fecha_vencimiento timestamp without time zone NOT NULL,
    proximo_aviso timestamp without time zone NOT NULL,
    notificado_email boolean DEFAULT false,
    activo boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.admin_tareas OWNER TO postgres;

--
-- Name: admin_tareas_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.admin_tareas_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.admin_tareas_id_seq OWNER TO postgres;

--
-- Name: admin_tareas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.admin_tareas_id_seq OWNED BY public.admin_tareas.id;


--
-- Name: afp_tasas_mensuales; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.afp_tasas_mensuales (
    id integer NOT NULL,
    afp_id integer,
    anio integer NOT NULL,
    mes integer NOT NULL,
    aporte_obligatorio numeric(5,4) DEFAULT 0.1000,
    comision_flujo numeric(5,4) NOT NULL,
    comision_mixta_flujo numeric(5,4) NOT NULL,
    prima_seguro numeric(5,4) NOT NULL,
    comision_anual_saldo numeric(5,4) DEFAULT 0
);


ALTER TABLE public.afp_tasas_mensuales OWNER TO postgres;

--
-- Name: afp_tasas_mensuales_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.afp_tasas_mensuales_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.afp_tasas_mensuales_id_seq OWNER TO postgres;

--
-- Name: afp_tasas_mensuales_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.afp_tasas_mensuales_id_seq OWNED BY public.afp_tasas_mensuales.id;


--
-- Name: afps; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.afps (
    id integer NOT NULL,
    codigo_sbs character varying(10),
    nombre character varying(50) NOT NULL,
    activo boolean DEFAULT true
);


ALTER TABLE public.afps OWNER TO postgres;

--
-- Name: afps_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.afps_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.afps_id_seq OWNER TO postgres;

--
-- Name: afps_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.afps_id_seq OWNED BY public.afps.id;


--
-- Name: base_regimen_default; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.base_regimen_default (
    id integer NOT NULL,
    concepto_calculado_id integer NOT NULL,
    regimen_id integer NOT NULL,
    concepto_modelo_id integer NOT NULL,
    variable_calculo character varying(50) NOT NULL,
    CONSTRAINT chk_variable_calculo_default CHECK (((variable_calculo)::text = ANY ((ARRAY['REMUNERACION_BASICA'::character varying, 'ASIGNACION_FAMILIAR'::character varying, 'SEXTO_GRATIFICACION'::character varying, 'REMUNERACION_VARIABLE'::character varying, 'REMUNERACION_COMPUTABLE'::character varying, 'MUC'::character varying, 'BET'::character varying, 'RETRIBUCION_MENSUAL'::character varying, 'VALORIZACION_PRINCIPAL'::character varying, 'VALORIZACION_AJUSTADA'::character varying])::text[])))
);


ALTER TABLE public.base_regimen_default OWNER TO postgres;

--
-- Name: base_regimen_default_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.base_regimen_default_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.base_regimen_default_id_seq OWNER TO postgres;

--
-- Name: base_regimen_default_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.base_regimen_default_id_seq OWNED BY public.base_regimen_default.id;


--
-- Name: base_regimen_tenant; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.base_regimen_tenant (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    concepto_calculado_id integer NOT NULL,
    regimen_id integer NOT NULL,
    concepto_tenant_id integer NOT NULL,
    variable_calculo character varying(50) NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    CONSTRAINT chk_variable_calculo_tenant CHECK (((variable_calculo)::text = ANY ((ARRAY['REMUNERACION_BASICA'::character varying, 'ASIGNACION_FAMILIAR'::character varying, 'SEXTO_GRATIFICACION'::character varying, 'REMUNERACION_VARIABLE'::character varying, 'REMUNERACION_COMPUTABLE'::character varying, 'MUC'::character varying, 'BET'::character varying, 'RETRIBUCION_MENSUAL'::character varying, 'VALORIZACION_PRINCIPAL'::character varying, 'VALORIZACION_AJUSTADA'::character varying])::text[])))
);


ALTER TABLE public.base_regimen_tenant OWNER TO postgres;

--
-- Name: base_regimen_tenant_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.base_regimen_tenant_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.base_regimen_tenant_id_seq OWNER TO postgres;

--
-- Name: base_regimen_tenant_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.base_regimen_tenant_id_seq OWNED BY public.base_regimen_tenant.id;


--
-- Name: clasificadores_mef; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.clasificadores_mef (
    id integer NOT NULL,
    anio integer DEFAULT 2026 NOT NULL,
    codigo character varying(50) NOT NULL,
    codigo_limpio character varying(50) NOT NULL,
    descripcion character varying(255) NOT NULL,
    nivel integer NOT NULL,
    tipo_transaccion character varying(50) NOT NULL,
    activo boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    parent_id integer
);


ALTER TABLE public.clasificadores_mef OWNER TO postgres;

--
-- Name: clasificadores_mef_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.clasificadores_mef_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.clasificadores_mef_id_seq OWNER TO postgres;

--
-- Name: clasificadores_mef_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.clasificadores_mef_id_seq OWNED BY public.clasificadores_mef.id;


--
-- Name: conceptos_afectaciones; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.conceptos_afectaciones (
    id integer NOT NULL,
    concepto_base_id integer NOT NULL,
    concepto_derivado_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.conceptos_afectaciones OWNER TO postgres;

--
-- Name: conceptos_afectaciones_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.conceptos_afectaciones_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.conceptos_afectaciones_id_seq OWNER TO postgres;

--
-- Name: conceptos_afectaciones_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.conceptos_afectaciones_id_seq OWNED BY public.conceptos_afectaciones.id;


--
-- Name: conceptos_calculados; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.conceptos_calculados (
    id integer NOT NULL,
    nombre character varying(150) NOT NULL,
    tipo character varying(50) NOT NULL,
    codigo_interno character varying(50) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.conceptos_calculados OWNER TO postgres;

--
-- Name: conceptos_calculados_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.conceptos_calculados_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.conceptos_calculados_id_seq OWNER TO postgres;

--
-- Name: conceptos_calculados_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.conceptos_calculados_id_seq OWNED BY public.conceptos_calculados.id;


--
-- Name: conceptos_maestros; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.conceptos_maestros (
    id integer NOT NULL,
    codigo character varying(50) NOT NULL,
    descripcion character varying(255) NOT NULL,
    tipo character varying(50) NOT NULL,
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    parent_id integer,
    codigo_interno character varying(50) NOT NULL,
    origen character varying(20) DEFAULT 'sunat'::character varying NOT NULL,
    CONSTRAINT chk_conceptos_maestros_origen CHECK (((origen)::text = ANY ((ARRAY['sunat'::character varying, 'interno'::character varying])::text[])))
);


ALTER TABLE public.conceptos_maestros OWNER TO postgres;

--
-- Name: conceptos_maestros_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.conceptos_maestros_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.conceptos_maestros_id_seq OWNER TO postgres;

--
-- Name: conceptos_maestros_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.conceptos_maestros_id_seq OWNED BY public.conceptos_maestros.id;


--
-- Name: conceptos_modelo; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.conceptos_modelo (
    id integer NOT NULL,
    concepto_id integer NOT NULL,
    nombre_personalizado character varying(150) NOT NULL,
    frecuencia_meses character varying(50) DEFAULT '1,2,3,4,5,6,7,8,9,10,11,12'::character varying,
    clasificador_id integer,
    es_extraordinario boolean DEFAULT false,
    requiere_monto boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    es_pensionable boolean DEFAULT false,
    es_remunerativa boolean DEFAULT false,
    es_base_cts boolean DEFAULT false,
    es_base_beneficios_sociales boolean DEFAULT false,
    es_ocasional boolean DEFAULT false NOT NULL,
    es_afecto_cargas_sociales boolean DEFAULT false NOT NULL
);


ALTER TABLE public.conceptos_modelo OWNER TO postgres;

--
-- Name: conceptos_modelo_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.conceptos_modelo_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.conceptos_modelo_id_seq OWNER TO postgres;

--
-- Name: conceptos_modelo_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.conceptos_modelo_id_seq OWNED BY public.conceptos_modelo.id;


--
-- Name: conceptos_tenant; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.conceptos_tenant (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    concepto_id integer NOT NULL,
    nombre_personalizado character varying(150) NOT NULL,
    frecuencia_meses character varying(50) DEFAULT '1,2,3,4,5,6,7,8,9,10,11,12'::character varying,
    activo boolean DEFAULT true,
    clasificador_id integer,
    es_extraordinario boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    requiere_monto boolean DEFAULT false,
    modelo_id integer,
    es_pensionable boolean DEFAULT false,
    es_remunerativa boolean DEFAULT false,
    es_base_cts boolean DEFAULT false,
    es_base_beneficios_sociales boolean DEFAULT false,
    es_ocasional boolean DEFAULT false NOT NULL,
    es_afecto_cargas_sociales boolean DEFAULT false NOT NULL
);


ALTER TABLE public.conceptos_tenant OWNER TO postgres;

--
-- Name: conceptos_tenant_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.conceptos_tenant_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.conceptos_tenant_id_seq OWNER TO postgres;

--
-- Name: conceptos_tenant_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.conceptos_tenant_id_seq OWNED BY public.conceptos_tenant.id;


--
-- Name: contrato_conceptos_snapshot; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.contrato_conceptos_snapshot (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    contrato_id integer NOT NULL,
    concepto_tenant_id integer NOT NULL,
    monto numeric(12,2) DEFAULT 0.00,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.contrato_conceptos_snapshot OWNER TO postgres;

--
-- Name: contrato_conceptos_snapshot_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.contrato_conceptos_snapshot_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.contrato_conceptos_snapshot_id_seq OWNER TO postgres;

--
-- Name: contrato_conceptos_snapshot_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.contrato_conceptos_snapshot_id_seq OWNED BY public.contrato_conceptos_snapshot.id;


--
-- Name: contratos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.contratos (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    trabajador_id integer NOT NULL,
    fecha_inicio date NOT NULL,
    fecha_fin date,
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    puesto_id integer NOT NULL,
    tipo_contrato character varying(100),
    nivel character varying(100),
    motivo_baja character varying(100)
);


ALTER TABLE public.contratos OWNER TO postgres;

--
-- Name: contratos_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.contratos_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.contratos_id_seq OWNER TO postgres;

--
-- Name: contratos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.contratos_id_seq OWNED BY public.contratos.id;


--
-- Name: fuentes_rubros; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fuentes_rubros (
    id integer NOT NULL,
    anio integer NOT NULL,
    fuente_financiamiento character varying(150) NOT NULL,
    rubro character varying(150) NOT NULL,
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    codigo_fuente_rubro character varying(20)
);


ALTER TABLE public.fuentes_rubros OWNER TO postgres;

--
-- Name: fuentes_rubros_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.fuentes_rubros_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.fuentes_rubros_id_seq OWNER TO postgres;

--
-- Name: fuentes_rubros_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.fuentes_rubros_id_seq OWNED BY public.fuentes_rubros.id;


--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.goose_db_version OWNER TO postgres;

--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

ALTER TABLE public.goose_db_version ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: liquidaciones_cese; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.liquidaciones_cese (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    contrato_id integer NOT NULL,
    fecha_inicio_computable date NOT NULL,
    fecha_cese date NOT NULL,
    motivo character varying(100),
    anos_servicios integer DEFAULT 0,
    meses_servicios integer DEFAULT 0,
    remuneracion_computable numeric(10,2) DEFAULT 0,
    monto_cts numeric(10,2) DEFAULT 0,
    monto_vacaciones_truncas numeric(10,2) DEFAULT 0,
    monto_gratificacion_trunca numeric(10,2) DEFAULT 0,
    total_liquidacion numeric(10,2) DEFAULT 0,
    estado character varying(20) DEFAULT 'BORRADOR'::character varying,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    monto_vacaciones_no_gozadas numeric(10,2) DEFAULT 0,
    monto_indemnizacion_vacacional numeric(10,2) DEFAULT 0,
    periodos_vencidos_vacaciones integer DEFAULT 0,
    periodos_no_vencidos_vacaciones integer DEFAULT 0,
    dias_servicios integer DEFAULT 0
);


ALTER TABLE public.liquidaciones_cese OWNER TO postgres;

--
-- Name: liquidaciones_cese_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.liquidaciones_cese_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.liquidaciones_cese_id_seq OWNER TO postgres;

--
-- Name: liquidaciones_cese_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.liquidaciones_cese_id_seq OWNED BY public.liquidaciones_cese.id;


--
-- Name: metas_presupuestales; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.metas_presupuestales (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    anio integer NOT NULL,
    codigo character varying(20) NOT NULL,
    descripcion character varying(512) NOT NULL,
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.metas_presupuestales OWNER TO postgres;

--
-- Name: metas_presupuestales_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.metas_presupuestales_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.metas_presupuestales_id_seq OWNER TO postgres;

--
-- Name: metas_presupuestales_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.metas_presupuestales_id_seq OWNED BY public.metas_presupuestales.id;


--
-- Name: notificaciones; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.notificaciones (
    id integer NOT NULL,
    tenant_id integer,
    usuario_id integer,
    titulo character varying(200) NOT NULL,
    mensaje text NOT NULL,
    tipo character varying(50) NOT NULL,
    leido boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.notificaciones OWNER TO postgres;

--
-- Name: notificaciones_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.notificaciones_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.notificaciones_id_seq OWNER TO postgres;

--
-- Name: notificaciones_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.notificaciones_id_seq OWNED BY public.notificaciones.id;


--
-- Name: ocurrencias_asistencia; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ocurrencias_asistencia (
    id integer NOT NULL,
    contrato_id integer NOT NULL,
    tipo character varying(20) NOT NULL,
    fecha_ocurrencia date NOT NULL,
    cantidad numeric(10,2) NOT NULL,
    procesado boolean DEFAULT false,
    planilla_id_descuento integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.ocurrencias_asistencia OWNER TO postgres;

--
-- Name: ocurrencias_asistencia_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ocurrencias_asistencia_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ocurrencias_asistencia_id_seq OWNER TO postgres;

--
-- Name: ocurrencias_asistencia_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.ocurrencias_asistencia_id_seq OWNED BY public.ocurrencias_asistencia.id;


--
-- Name: organigramas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organigramas (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    documento_aprobacion character varying(200) NOT NULL,
    descripcion character varying(255),
    fecha_vigencia date NOT NULL,
    activo boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.organigramas OWNER TO postgres;

--
-- Name: organigramas_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.organigramas_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.organigramas_id_seq OWNER TO postgres;

--
-- Name: organigramas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.organigramas_id_seq OWNED BY public.organigramas.id;


--
-- Name: pap_detalles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pap_detalles (
    id integer NOT NULL,
    version_id integer,
    meta_codigo character varying(50),
    meta_descripcion character varying(255),
    fuente_rubro_codigo character varying(50),
    fuente_rubro_descripcion character varying(255),
    clasificador_codigo_limpio character varying(50),
    clasificador_descripcion character varying(255),
    mes_01 numeric(10,2) DEFAULT 0,
    mes_02 numeric(10,2) DEFAULT 0,
    mes_03 numeric(10,2) DEFAULT 0,
    mes_04 numeric(10,2) DEFAULT 0,
    mes_05 numeric(10,2) DEFAULT 0,
    mes_06 numeric(10,2) DEFAULT 0,
    mes_07 numeric(10,2) DEFAULT 0,
    mes_08 numeric(10,2) DEFAULT 0,
    mes_09 numeric(10,2) DEFAULT 0,
    mes_10 numeric(10,2) DEFAULT 0,
    mes_11 numeric(10,2) DEFAULT 0,
    mes_12 numeric(10,2) DEFAULT 0,
    total_anual numeric(12,2) DEFAULT 0
);


ALTER TABLE public.pap_detalles OWNER TO postgres;

--
-- Name: pap_detalles_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.pap_detalles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.pap_detalles_id_seq OWNER TO postgres;

--
-- Name: pap_detalles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.pap_detalles_id_seq OWNED BY public.pap_detalles.id;


--
-- Name: pap_versiones; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pap_versiones (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    anio integer NOT NULL,
    tipo character varying(50) NOT NULL,
    fecha_generacion timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    estado character varying(20) DEFAULT 'CERRADA'::character varying
);


ALTER TABLE public.pap_versiones OWNER TO postgres;

--
-- Name: pap_versiones_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.pap_versiones_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.pap_versiones_id_seq OWNER TO postgres;

--
-- Name: pap_versiones_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.pap_versiones_id_seq OWNED BY public.pap_versiones.id;


--
-- Name: parametros_globales; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.parametros_globales (
    id integer NOT NULL,
    clave character varying(50) NOT NULL,
    valor numeric(15,4) NOT NULL,
    descripcion character varying(255),
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    fecha_desde date NOT NULL,
    fecha_hasta date
);


ALTER TABLE public.parametros_globales OWNER TO postgres;

--
-- Name: parametros_globales_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.parametros_globales_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.parametros_globales_id_seq OWNER TO postgres;

--
-- Name: parametros_globales_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.parametros_globales_id_seq OWNED BY public.parametros_globales.id;


--
-- Name: planilla_conceptos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.planilla_conceptos (
    id integer NOT NULL,
    planilla_detalle_id integer NOT NULL,
    concepto_tenant_id integer,
    tipo_concepto character varying(20) NOT NULL,
    monto numeric(10,2) NOT NULL,
    maestro_id integer DEFAULT 0 NOT NULL,
    codigo_sunat character varying(10),
    nombre_en_boleta character varying(150)
);


ALTER TABLE public.planilla_conceptos OWNER TO postgres;

--
-- Name: planilla_conceptos_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.planilla_conceptos_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.planilla_conceptos_id_seq OWNER TO postgres;

--
-- Name: planilla_conceptos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.planilla_conceptos_id_seq OWNED BY public.planilla_conceptos.id;


--
-- Name: planilla_cts_detalles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.planilla_cts_detalles (
    id integer NOT NULL,
    planilla_cts_id integer NOT NULL,
    contrato_id integer NOT NULL,
    sueldo_basico numeric(10,2) DEFAULT 0,
    asignacion_familiar numeric(10,2) DEFAULT 0,
    sexto_gratificacion numeric(10,2) DEFAULT 0,
    promedio_variables numeric(10,2) DEFAULT 0,
    remuneracion_computable numeric(10,2) DEFAULT 0,
    meses_computables integer DEFAULT 0,
    dias_faltas integer DEFAULT 0,
    monto_descuento_faltas numeric(10,2) DEFAULT 0,
    monto_cts numeric(10,2) DEFAULT 0,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.planilla_cts_detalles OWNER TO postgres;

--
-- Name: planilla_cts_detalles_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.planilla_cts_detalles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.planilla_cts_detalles_id_seq OWNER TO postgres;

--
-- Name: planilla_cts_detalles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.planilla_cts_detalles_id_seq OWNED BY public.planilla_cts_detalles.id;


--
-- Name: planilla_detalles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.planilla_detalles (
    id integer NOT NULL,
    planilla_id integer NOT NULL,
    contrato_id integer NOT NULL,
    total_ingresos numeric(10,2) DEFAULT 0.00,
    total_retenciones numeric(10,2) DEFAULT 0.00,
    total_aportes numeric(10,2) DEFAULT 0.00,
    neto_pagar numeric(10,2) DEFAULT 0.00,
    trabajador_nombre_completo character varying(250),
    trabajador_numero_documento character varying(20),
    puesto_codigo_airhsp character varying(50),
    puesto_nombre character varying(200),
    organigrama_documento_aprobacion character varying(200),
    unidad_organica_nombre character varying(200),
    unidad_organica_tipo character varying(50),
    sueldo_basico_historico numeric(10,2)
);


ALTER TABLE public.planilla_detalles OWNER TO postgres;

--
-- Name: planilla_detalles_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.planilla_detalles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.planilla_detalles_id_seq OWNER TO postgres;

--
-- Name: planilla_detalles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.planilla_detalles_id_seq OWNED BY public.planilla_detalles.id;


--
-- Name: planillas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.planillas (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    anio integer NOT NULL,
    mes integer NOT NULL,
    descripcion character varying(255) NOT NULL,
    estado character varying(20) DEFAULT 'BORRADOR'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.planillas OWNER TO postgres;

--
-- Name: planillas_cts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.planillas_cts (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    anio integer NOT NULL,
    periodo character varying(20) NOT NULL,
    estado character varying(20) DEFAULT 'BORRADOR'::character varying,
    fecha_calculo timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.planillas_cts OWNER TO postgres;

--
-- Name: planillas_cts_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.planillas_cts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.planillas_cts_id_seq OWNER TO postgres;

--
-- Name: planillas_cts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.planillas_cts_id_seq OWNED BY public.planillas_cts.id;


--
-- Name: planillas_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.planillas_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.planillas_id_seq OWNER TO postgres;

--
-- Name: planillas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.planillas_id_seq OWNED BY public.planillas.id;


--
-- Name: puesto_conceptos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.puesto_conceptos (
    id integer NOT NULL,
    puesto_id integer NOT NULL,
    concepto_tenant_id integer NOT NULL,
    monto numeric(10,2),
    activo boolean DEFAULT true
);


ALTER TABLE public.puesto_conceptos OWNER TO postgres;

--
-- Name: puesto_conceptos_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.puesto_conceptos_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.puesto_conceptos_id_seq OWNER TO postgres;

--
-- Name: puesto_conceptos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.puesto_conceptos_id_seq OWNED BY public.puesto_conceptos.id;


--
-- Name: puestos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.puestos (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    meta_id integer,
    fuente_rubro_id integer,
    regimen_id integer NOT NULL,
    nombre character varying(150) NOT NULL,
    sueldo_presupuestado numeric(10,2) NOT NULL,
    estado character varying(20) DEFAULT 'VACANTE'::character varying,
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    es_dietario boolean DEFAULT false,
    unidad_organica_id integer,
    codigo_airhsp character varying(50)
);


ALTER TABLE public.puestos OWNER TO postgres;

--
-- Name: puestos_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.puestos_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.puestos_id_seq OWNER TO postgres;

--
-- Name: puestos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.puestos_id_seq OWNED BY public.puestos.id;


--
-- Name: regimen_concepto_modelo; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.regimen_concepto_modelo (
    regimen_id integer NOT NULL,
    concepto_modelo_id integer NOT NULL
);


ALTER TABLE public.regimen_concepto_modelo OWNER TO postgres;

--
-- Name: regimen_concepto_tenant; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.regimen_concepto_tenant (
    tenant_id integer NOT NULL,
    regimen_id integer NOT NULL,
    concepto_tenant_id integer NOT NULL
);


ALTER TABLE public.regimen_concepto_tenant OWNER TO postgres;

--
-- Name: regimenes_laborales; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.regimenes_laborales (
    id integer NOT NULL,
    codigo character varying(10) NOT NULL,
    descripcion character varying(150) NOT NULL
);


ALTER TABLE public.regimenes_laborales OWNER TO postgres;

--
-- Name: regimenes_laborales_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.regimenes_laborales_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.regimenes_laborales_id_seq OWNER TO postgres;

--
-- Name: regimenes_laborales_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.regimenes_laborales_id_seq OWNED BY public.regimenes_laborales.id;


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tenants (
    id integer NOT NULL,
    nombre character varying(255) NOT NULL,
    ruc character varying(11) NOT NULL,
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    direccion character varying(255),
    frase_gestion character varying(255),
    logo_url character varying(255),
    slug character varying(100),
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.tenants OWNER TO postgres;

--
-- Name: tenants_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.tenants_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.tenants_id_seq OWNER TO postgres;

--
-- Name: tenants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.tenants_id_seq OWNED BY public.tenants.id;


--
-- Name: trabajadores; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.trabajadores (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    tipo_documento character varying(20) DEFAULT 'DNI'::character varying NOT NULL,
    numero_documento character varying(20) NOT NULL,
    nombres character varying(100) NOT NULL,
    apellido_paterno character varying(100) NOT NULL,
    apellido_materno character varying(100) NOT NULL,
    fecha_nacimiento date,
    sexo character varying(1),
    activo boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    regimen_pensionario character varying(20) DEFAULT 'ONP'::character varying,
    afp_id integer,
    afp_tipo_comision character varying(10),
    cuspp character varying(20),
    fecha_ingreso date,
    fecha_cese date,
    direccion character varying(255),
    banco character varying(100),
    cuenta character varying(50),
    cci character varying(50)
);


ALTER TABLE public.trabajadores OWNER TO postgres;

--
-- Name: trabajadores_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.trabajadores_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.trabajadores_id_seq OWNER TO postgres;

--
-- Name: trabajadores_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.trabajadores_id_seq OWNED BY public.trabajadores.id;


--
-- Name: unidades_organicas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.unidades_organicas (
    id integer NOT NULL,
    tenant_id integer NOT NULL,
    organigrama_id integer NOT NULL,
    parent_id integer,
    codigo_mef character varying(50),
    nombre character varying(200) NOT NULL,
    tipo character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.unidades_organicas OWNER TO postgres;

--
-- Name: unidades_organicas_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.unidades_organicas_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.unidades_organicas_id_seq OWNER TO postgres;

--
-- Name: unidades_organicas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.unidades_organicas_id_seq OWNED BY public.unidades_organicas.id;


--
-- Name: usuarios; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.usuarios (
    id integer NOT NULL,
    tenant_id integer,
    nombre character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    rol character varying(50) NOT NULL,
    activo boolean DEFAULT true
);


ALTER TABLE public.usuarios OWNER TO postgres;

--
-- Name: usuarios_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.usuarios_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.usuarios_id_seq OWNER TO postgres;

--
-- Name: usuarios_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.usuarios_id_seq OWNED BY public.usuarios.id;


--
-- Name: admin_tareas id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_tareas ALTER COLUMN id SET DEFAULT nextval('public.admin_tareas_id_seq'::regclass);


--
-- Name: afp_tasas_mensuales id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afp_tasas_mensuales ALTER COLUMN id SET DEFAULT nextval('public.afp_tasas_mensuales_id_seq'::regclass);


--
-- Name: afps id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afps ALTER COLUMN id SET DEFAULT nextval('public.afps_id_seq'::regclass);


--
-- Name: base_regimen_default id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_default ALTER COLUMN id SET DEFAULT nextval('public.base_regimen_default_id_seq'::regclass);


--
-- Name: base_regimen_tenant id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant ALTER COLUMN id SET DEFAULT nextval('public.base_regimen_tenant_id_seq'::regclass);


--
-- Name: clasificadores_mef id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.clasificadores_mef ALTER COLUMN id SET DEFAULT nextval('public.clasificadores_mef_id_seq'::regclass);


--
-- Name: conceptos_afectaciones id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_afectaciones ALTER COLUMN id SET DEFAULT nextval('public.conceptos_afectaciones_id_seq'::regclass);


--
-- Name: conceptos_calculados id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_calculados ALTER COLUMN id SET DEFAULT nextval('public.conceptos_calculados_id_seq'::regclass);


--
-- Name: conceptos_maestros id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_maestros ALTER COLUMN id SET DEFAULT nextval('public.conceptos_maestros_id_seq'::regclass);


--
-- Name: conceptos_modelo id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_modelo ALTER COLUMN id SET DEFAULT nextval('public.conceptos_modelo_id_seq'::regclass);


--
-- Name: conceptos_tenant id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant ALTER COLUMN id SET DEFAULT nextval('public.conceptos_tenant_id_seq'::regclass);


--
-- Name: contrato_conceptos_snapshot id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contrato_conceptos_snapshot ALTER COLUMN id SET DEFAULT nextval('public.contrato_conceptos_snapshot_id_seq'::regclass);


--
-- Name: contratos id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contratos ALTER COLUMN id SET DEFAULT nextval('public.contratos_id_seq'::regclass);


--
-- Name: fuentes_rubros id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.fuentes_rubros ALTER COLUMN id SET DEFAULT nextval('public.fuentes_rubros_id_seq'::regclass);


--
-- Name: liquidaciones_cese id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.liquidaciones_cese ALTER COLUMN id SET DEFAULT nextval('public.liquidaciones_cese_id_seq'::regclass);


--
-- Name: metas_presupuestales id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.metas_presupuestales ALTER COLUMN id SET DEFAULT nextval('public.metas_presupuestales_id_seq'::regclass);


--
-- Name: notificaciones id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notificaciones ALTER COLUMN id SET DEFAULT nextval('public.notificaciones_id_seq'::regclass);


--
-- Name: ocurrencias_asistencia id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ocurrencias_asistencia ALTER COLUMN id SET DEFAULT nextval('public.ocurrencias_asistencia_id_seq'::regclass);


--
-- Name: organigramas id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.organigramas ALTER COLUMN id SET DEFAULT nextval('public.organigramas_id_seq'::regclass);


--
-- Name: pap_detalles id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pap_detalles ALTER COLUMN id SET DEFAULT nextval('public.pap_detalles_id_seq'::regclass);


--
-- Name: pap_versiones id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pap_versiones ALTER COLUMN id SET DEFAULT nextval('public.pap_versiones_id_seq'::regclass);


--
-- Name: parametros_globales id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.parametros_globales ALTER COLUMN id SET DEFAULT nextval('public.parametros_globales_id_seq'::regclass);


--
-- Name: planilla_conceptos id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_conceptos ALTER COLUMN id SET DEFAULT nextval('public.planilla_conceptos_id_seq'::regclass);


--
-- Name: planilla_cts_detalles id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_cts_detalles ALTER COLUMN id SET DEFAULT nextval('public.planilla_cts_detalles_id_seq'::regclass);


--
-- Name: planilla_detalles id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_detalles ALTER COLUMN id SET DEFAULT nextval('public.planilla_detalles_id_seq'::regclass);


--
-- Name: planillas id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas ALTER COLUMN id SET DEFAULT nextval('public.planillas_id_seq'::regclass);


--
-- Name: planillas_cts id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas_cts ALTER COLUMN id SET DEFAULT nextval('public.planillas_cts_id_seq'::regclass);


--
-- Name: puesto_conceptos id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puesto_conceptos ALTER COLUMN id SET DEFAULT nextval('public.puesto_conceptos_id_seq'::regclass);


--
-- Name: puestos id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos ALTER COLUMN id SET DEFAULT nextval('public.puestos_id_seq'::regclass);


--
-- Name: regimenes_laborales id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimenes_laborales ALTER COLUMN id SET DEFAULT nextval('public.regimenes_laborales_id_seq'::regclass);


--
-- Name: tenants id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tenants ALTER COLUMN id SET DEFAULT nextval('public.tenants_id_seq'::regclass);


--
-- Name: trabajadores id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trabajadores ALTER COLUMN id SET DEFAULT nextval('public.trabajadores_id_seq'::regclass);


--
-- Name: unidades_organicas id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unidades_organicas ALTER COLUMN id SET DEFAULT nextval('public.unidades_organicas_id_seq'::regclass);


--
-- Name: usuarios id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.usuarios ALTER COLUMN id SET DEFAULT nextval('public.usuarios_id_seq'::regclass);


--
-- Name: admin_tareas admin_tareas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_tareas
    ADD CONSTRAINT admin_tareas_pkey PRIMARY KEY (id);


--
-- Name: afp_tasas_mensuales afp_tasas_mensuales_afp_id_anio_mes_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afp_tasas_mensuales
    ADD CONSTRAINT afp_tasas_mensuales_afp_id_anio_mes_key UNIQUE (afp_id, anio, mes);


--
-- Name: afp_tasas_mensuales afp_tasas_mensuales_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afp_tasas_mensuales
    ADD CONSTRAINT afp_tasas_mensuales_pkey PRIMARY KEY (id);


--
-- Name: afps afps_codigo_sbs_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afps
    ADD CONSTRAINT afps_codigo_sbs_key UNIQUE (codigo_sbs);


--
-- Name: afps afps_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afps
    ADD CONSTRAINT afps_pkey PRIMARY KEY (id);


--
-- Name: base_regimen_default base_regimen_default_concepto_calculado_id_regimen_id_conce_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_default
    ADD CONSTRAINT base_regimen_default_concepto_calculado_id_regimen_id_conce_key UNIQUE (concepto_calculado_id, regimen_id, concepto_modelo_id, variable_calculo);


--
-- Name: base_regimen_default base_regimen_default_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_default
    ADD CONSTRAINT base_regimen_default_pkey PRIMARY KEY (id);


--
-- Name: base_regimen_tenant base_regimen_tenant_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant
    ADD CONSTRAINT base_regimen_tenant_pkey PRIMARY KEY (id);


--
-- Name: base_regimen_tenant base_regimen_tenant_tenant_id_concepto_calculado_id_regimen_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant
    ADD CONSTRAINT base_regimen_tenant_tenant_id_concepto_calculado_id_regimen_key UNIQUE (tenant_id, concepto_calculado_id, regimen_id, concepto_tenant_id, variable_calculo);


--
-- Name: clasificadores_mef clasificadores_mef_codigo_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.clasificadores_mef
    ADD CONSTRAINT clasificadores_mef_codigo_key UNIQUE (codigo);


--
-- Name: clasificadores_mef clasificadores_mef_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.clasificadores_mef
    ADD CONSTRAINT clasificadores_mef_pkey PRIMARY KEY (id);


--
-- Name: conceptos_afectaciones conceptos_afectaciones_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_afectaciones
    ADD CONSTRAINT conceptos_afectaciones_pkey PRIMARY KEY (id);


--
-- Name: conceptos_calculados conceptos_calculados_codigo_interno_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_calculados
    ADD CONSTRAINT conceptos_calculados_codigo_interno_key UNIQUE (codigo_interno);


--
-- Name: conceptos_calculados conceptos_calculados_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_calculados
    ADD CONSTRAINT conceptos_calculados_pkey PRIMARY KEY (id);


--
-- Name: conceptos_maestros conceptos_maestros_codigo_interno_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_maestros
    ADD CONSTRAINT conceptos_maestros_codigo_interno_key UNIQUE (codigo_interno);


--
-- Name: conceptos_maestros conceptos_maestros_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_maestros
    ADD CONSTRAINT conceptos_maestros_pkey PRIMARY KEY (id);


--
-- Name: conceptos_modelo conceptos_modelo_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_modelo
    ADD CONSTRAINT conceptos_modelo_pkey PRIMARY KEY (id);


--
-- Name: conceptos_tenant conceptos_tenant_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT conceptos_tenant_pkey PRIMARY KEY (id);


--
-- Name: contrato_conceptos_snapshot contrato_conceptos_snapshot_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contrato_conceptos_snapshot
    ADD CONSTRAINT contrato_conceptos_snapshot_pkey PRIMARY KEY (id);


--
-- Name: contratos contratos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contratos
    ADD CONSTRAINT contratos_pkey PRIMARY KEY (id);


--
-- Name: fuentes_rubros fuentes_rubros_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.fuentes_rubros
    ADD CONSTRAINT fuentes_rubros_pkey PRIMARY KEY (id);


--
-- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);


--
-- Name: liquidaciones_cese liquidaciones_cese_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.liquidaciones_cese
    ADD CONSTRAINT liquidaciones_cese_pkey PRIMARY KEY (id);


--
-- Name: metas_presupuestales metas_presupuestales_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.metas_presupuestales
    ADD CONSTRAINT metas_presupuestales_pkey PRIMARY KEY (id);


--
-- Name: notificaciones notificaciones_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notificaciones
    ADD CONSTRAINT notificaciones_pkey PRIMARY KEY (id);


--
-- Name: ocurrencias_asistencia ocurrencias_asistencia_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ocurrencias_asistencia
    ADD CONSTRAINT ocurrencias_asistencia_pkey PRIMARY KEY (id);


--
-- Name: organigramas organigramas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.organigramas
    ADD CONSTRAINT organigramas_pkey PRIMARY KEY (id);


--
-- Name: pap_detalles pap_detalles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pap_detalles
    ADD CONSTRAINT pap_detalles_pkey PRIMARY KEY (id);


--
-- Name: pap_versiones pap_versiones_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pap_versiones
    ADD CONSTRAINT pap_versiones_pkey PRIMARY KEY (id);


--
-- Name: parametros_globales parametros_globales_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.parametros_globales
    ADD CONSTRAINT parametros_globales_pkey PRIMARY KEY (id);


--
-- Name: planilla_conceptos planilla_conceptos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_conceptos
    ADD CONSTRAINT planilla_conceptos_pkey PRIMARY KEY (id);


--
-- Name: planilla_cts_detalles planilla_cts_detalles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_cts_detalles
    ADD CONSTRAINT planilla_cts_detalles_pkey PRIMARY KEY (id);


--
-- Name: planilla_detalles planilla_detalles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_detalles
    ADD CONSTRAINT planilla_detalles_pkey PRIMARY KEY (id);


--
-- Name: planillas_cts planillas_cts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas_cts
    ADD CONSTRAINT planillas_cts_pkey PRIMARY KEY (id);


--
-- Name: planillas planillas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas
    ADD CONSTRAINT planillas_pkey PRIMARY KEY (id);


--
-- Name: puesto_conceptos puesto_conceptos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puesto_conceptos
    ADD CONSTRAINT puesto_conceptos_pkey PRIMARY KEY (id);


--
-- Name: puestos puestos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos
    ADD CONSTRAINT puestos_pkey PRIMARY KEY (id);


--
-- Name: regimen_concepto_modelo regimen_concepto_modelo_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_modelo
    ADD CONSTRAINT regimen_concepto_modelo_pkey PRIMARY KEY (regimen_id, concepto_modelo_id);


--
-- Name: regimen_concepto_tenant regimen_concepto_tenant_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_tenant
    ADD CONSTRAINT regimen_concepto_tenant_pkey PRIMARY KEY (tenant_id, regimen_id, concepto_tenant_id);


--
-- Name: regimenes_laborales regimenes_laborales_codigo_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimenes_laborales
    ADD CONSTRAINT regimenes_laborales_codigo_key UNIQUE (codigo);


--
-- Name: regimenes_laborales regimenes_laborales_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimenes_laborales
    ADD CONSTRAINT regimenes_laborales_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: tenants tenants_ruc_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_ruc_key UNIQUE (ruc);


--
-- Name: tenants tenants_slug_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_slug_key UNIQUE (slug);


--
-- Name: trabajadores trabajadores_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT trabajadores_pkey PRIMARY KEY (id);


--
-- Name: unidades_organicas unidades_organicas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unidades_organicas
    ADD CONSTRAINT unidades_organicas_pkey PRIMARY KEY (id);


--
-- Name: conceptos_afectaciones unique_afectacion; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_afectaciones
    ADD CONSTRAINT unique_afectacion UNIQUE (concepto_base_id, concepto_derivado_id);


--
-- Name: clasificadores_mef unique_anio_codigo; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.clasificadores_mef
    ADD CONSTRAINT unique_anio_codigo UNIQUE (anio, codigo_limpio);


--
-- Name: parametros_globales unique_clave_fecha; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.parametros_globales
    ADD CONSTRAINT unique_clave_fecha UNIQUE (clave, fecha_desde);


--
-- Name: contrato_conceptos_snapshot unique_contrato_concepto_snapshot; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contrato_conceptos_snapshot
    ADD CONSTRAINT unique_contrato_concepto_snapshot UNIQUE (contrato_id, concepto_tenant_id);


--
-- Name: planilla_detalles unique_detalle_contrato; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_detalles
    ADD CONSTRAINT unique_detalle_contrato UNIQUE (planilla_id, contrato_id);


--
-- Name: trabajadores unique_documento_tenant; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT unique_documento_tenant UNIQUE (tenant_id, tipo_documento, numero_documento);


--
-- Name: fuentes_rubros unique_fuente_rubro_anio; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.fuentes_rubros
    ADD CONSTRAINT unique_fuente_rubro_anio UNIQUE (anio, fuente_financiamiento, rubro);


--
-- Name: metas_presupuestales unique_meta_anio_tenant; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.metas_presupuestales
    ADD CONSTRAINT unique_meta_anio_tenant UNIQUE (tenant_id, anio, codigo);


--
-- Name: conceptos_modelo unique_nombre_modelo; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_modelo
    ADD CONSTRAINT unique_nombre_modelo UNIQUE (nombre_personalizado);


--
-- Name: unidades_organicas unique_organigrama_nombre; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unidades_organicas
    ADD CONSTRAINT unique_organigrama_nombre UNIQUE (organigrama_id, nombre);


--
-- Name: planillas unique_planilla_mes; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas
    ADD CONSTRAINT unique_planilla_mes UNIQUE (tenant_id, anio, mes, descripcion);


--
-- Name: puesto_conceptos unique_puesto_concepto; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puesto_conceptos
    ADD CONSTRAINT unique_puesto_concepto UNIQUE (puesto_id, concepto_tenant_id);


--
-- Name: conceptos_tenant unique_tenant_id_id; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT unique_tenant_id_id UNIQUE (tenant_id, id);


--
-- Name: conceptos_tenant unique_tenant_modelo_id; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT unique_tenant_modelo_id UNIQUE (tenant_id, modelo_id);


--
-- Name: planillas_cts uq_planilla_cts_periodo; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas_cts
    ADD CONSTRAINT uq_planilla_cts_periodo UNIQUE (tenant_id, anio, periodo);


--
-- Name: usuarios usuarios_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_email_key UNIQUE (email);


--
-- Name: usuarios usuarios_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_pkey PRIMARY KEY (id);


--
-- Name: idx_admin_tareas_aviso; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_tareas_aviso ON public.admin_tareas USING btree (proximo_aviso) WHERE (activo = true);


--
-- Name: idx_base_regimen_tenant_calc; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_base_regimen_tenant_calc ON public.base_regimen_tenant USING btree (tenant_id, regimen_id, concepto_calculado_id) WHERE (activo = true);


--
-- Name: idx_conceptos_tenant_tid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_conceptos_tenant_tid ON public.conceptos_tenant USING btree (tenant_id);


--
-- Name: idx_contrato_conceptos_snapshot_cid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_contrato_conceptos_snapshot_cid ON public.contrato_conceptos_snapshot USING btree (contrato_id);


--
-- Name: idx_contratos_puesto; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_contratos_puesto ON public.contratos USING btree (puesto_id);


--
-- Name: idx_contratos_tenant; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_contratos_tenant ON public.contratos USING btree (tenant_id);


--
-- Name: idx_contratos_trabajador; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_contratos_trabajador ON public.contratos USING btree (trabajador_id);


--
-- Name: idx_metas_tenant_anio; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_metas_tenant_anio ON public.metas_presupuestales USING btree (tenant_id, anio);


--
-- Name: idx_notificaciones_tenant_leido; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_notificaciones_tenant_leido ON public.notificaciones USING btree (tenant_id, leido);


--
-- Name: idx_notificaciones_usuario_leido; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_notificaciones_usuario_leido ON public.notificaciones USING btree (usuario_id, leido);


--
-- Name: idx_organigramas_tenant; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organigramas_tenant ON public.organigramas USING btree (tenant_id);


--
-- Name: idx_planillas_tenant; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_planillas_tenant ON public.planillas USING btree (tenant_id, anio, mes);


--
-- Name: idx_puesto_conceptos_pid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_puesto_conceptos_pid ON public.puesto_conceptos USING btree (puesto_id);


--
-- Name: idx_puestos_tenant; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_puestos_tenant ON public.puestos USING btree (tenant_id);


--
-- Name: idx_regimen_concepto_tenant_lookup; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_regimen_concepto_tenant_lookup ON public.regimen_concepto_tenant USING btree (tenant_id, regimen_id);


--
-- Name: idx_trabajadores_tenant; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_trabajadores_tenant ON public.trabajadores USING btree (tenant_id);


--
-- Name: idx_unidades_organicas_org; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_unidades_organicas_org ON public.unidades_organicas USING btree (organigrama_id);


--
-- Name: afp_tasas_mensuales afp_tasas_mensuales_afp_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.afp_tasas_mensuales
    ADD CONSTRAINT afp_tasas_mensuales_afp_id_fkey FOREIGN KEY (afp_id) REFERENCES public.afps(id);


--
-- Name: base_regimen_default base_regimen_default_concepto_calculado_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_default
    ADD CONSTRAINT base_regimen_default_concepto_calculado_id_fkey FOREIGN KEY (concepto_calculado_id) REFERENCES public.conceptos_calculados(id) ON DELETE CASCADE;


--
-- Name: base_regimen_default base_regimen_default_concepto_modelo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_default
    ADD CONSTRAINT base_regimen_default_concepto_modelo_id_fkey FOREIGN KEY (concepto_modelo_id) REFERENCES public.conceptos_modelo(id) ON DELETE CASCADE;


--
-- Name: base_regimen_default base_regimen_default_regimen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_default
    ADD CONSTRAINT base_regimen_default_regimen_id_fkey FOREIGN KEY (regimen_id) REFERENCES public.regimenes_laborales(id) ON DELETE CASCADE;


--
-- Name: base_regimen_tenant base_regimen_tenant_concepto_calculado_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant
    ADD CONSTRAINT base_regimen_tenant_concepto_calculado_id_fkey FOREIGN KEY (concepto_calculado_id) REFERENCES public.conceptos_calculados(id) ON DELETE CASCADE;


--
-- Name: base_regimen_tenant base_regimen_tenant_regimen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant
    ADD CONSTRAINT base_regimen_tenant_regimen_id_fkey FOREIGN KEY (regimen_id) REFERENCES public.regimenes_laborales(id) ON DELETE CASCADE;


--
-- Name: base_regimen_tenant base_regimen_tenant_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant
    ADD CONSTRAINT base_regimen_tenant_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: clasificadores_mef clasificadores_mef_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.clasificadores_mef
    ADD CONSTRAINT clasificadores_mef_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.clasificadores_mef(id) ON DELETE SET NULL;


--
-- Name: conceptos_afectaciones conceptos_afectaciones_concepto_base_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_afectaciones
    ADD CONSTRAINT conceptos_afectaciones_concepto_base_id_fkey FOREIGN KEY (concepto_base_id) REFERENCES public.conceptos_maestros(id) ON DELETE CASCADE;


--
-- Name: conceptos_afectaciones conceptos_afectaciones_concepto_derivado_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_afectaciones
    ADD CONSTRAINT conceptos_afectaciones_concepto_derivado_id_fkey FOREIGN KEY (concepto_derivado_id) REFERENCES public.conceptos_maestros(id) ON DELETE CASCADE;


--
-- Name: conceptos_maestros conceptos_maestros_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_maestros
    ADD CONSTRAINT conceptos_maestros_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.conceptos_maestros(id) ON DELETE SET NULL;


--
-- Name: conceptos_modelo conceptos_modelo_clasificador_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_modelo
    ADD CONSTRAINT conceptos_modelo_clasificador_id_fkey FOREIGN KEY (clasificador_id) REFERENCES public.clasificadores_mef(id) ON DELETE SET NULL;


--
-- Name: conceptos_modelo conceptos_modelo_concepto_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_modelo
    ADD CONSTRAINT conceptos_modelo_concepto_id_fkey FOREIGN KEY (concepto_id) REFERENCES public.conceptos_maestros(id);


--
-- Name: conceptos_tenant conceptos_tenant_clasificador_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT conceptos_tenant_clasificador_id_fkey FOREIGN KEY (clasificador_id) REFERENCES public.clasificadores_mef(id) ON DELETE SET NULL;


--
-- Name: conceptos_tenant conceptos_tenant_concepto_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT conceptos_tenant_concepto_id_fkey FOREIGN KEY (concepto_id) REFERENCES public.conceptos_maestros(id);


--
-- Name: conceptos_tenant conceptos_tenant_modelo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT conceptos_tenant_modelo_id_fkey FOREIGN KEY (modelo_id) REFERENCES public.conceptos_modelo(id) ON DELETE CASCADE;


--
-- Name: conceptos_tenant conceptos_tenant_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.conceptos_tenant
    ADD CONSTRAINT conceptos_tenant_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: contrato_conceptos_snapshot contrato_conceptos_snapshot_concepto_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contrato_conceptos_snapshot
    ADD CONSTRAINT contrato_conceptos_snapshot_concepto_tenant_id_fkey FOREIGN KEY (concepto_tenant_id) REFERENCES public.conceptos_tenant(id) ON DELETE CASCADE;


--
-- Name: contrato_conceptos_snapshot contrato_conceptos_snapshot_contrato_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contrato_conceptos_snapshot
    ADD CONSTRAINT contrato_conceptos_snapshot_contrato_id_fkey FOREIGN KEY (contrato_id) REFERENCES public.contratos(id) ON DELETE CASCADE;


--
-- Name: contrato_conceptos_snapshot contrato_conceptos_snapshot_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contrato_conceptos_snapshot
    ADD CONSTRAINT contrato_conceptos_snapshot_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: contratos contratos_puesto_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contratos
    ADD CONSTRAINT contratos_puesto_id_fkey FOREIGN KEY (puesto_id) REFERENCES public.puestos(id);


--
-- Name: contratos contratos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contratos
    ADD CONSTRAINT contratos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: contratos contratos_trabajador_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contratos
    ADD CONSTRAINT contratos_trabajador_id_fkey FOREIGN KEY (trabajador_id) REFERENCES public.trabajadores(id) ON DELETE CASCADE;


--
-- Name: base_regimen_tenant fk_base_regimen_tenant_concepto; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_regimen_tenant
    ADD CONSTRAINT fk_base_regimen_tenant_concepto FOREIGN KEY (tenant_id, concepto_tenant_id) REFERENCES public.conceptos_tenant(tenant_id, id) ON DELETE CASCADE;


--
-- Name: liquidaciones_cese liquidaciones_cese_contrato_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.liquidaciones_cese
    ADD CONSTRAINT liquidaciones_cese_contrato_id_fkey FOREIGN KEY (contrato_id) REFERENCES public.contratos(id) ON DELETE CASCADE;


--
-- Name: metas_presupuestales metas_presupuestales_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.metas_presupuestales
    ADD CONSTRAINT metas_presupuestales_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: notificaciones notificaciones_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notificaciones
    ADD CONSTRAINT notificaciones_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: notificaciones notificaciones_usuario_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notificaciones
    ADD CONSTRAINT notificaciones_usuario_id_fkey FOREIGN KEY (usuario_id) REFERENCES public.usuarios(id) ON DELETE CASCADE;


--
-- Name: ocurrencias_asistencia ocurrencias_asistencia_contrato_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ocurrencias_asistencia
    ADD CONSTRAINT ocurrencias_asistencia_contrato_id_fkey FOREIGN KEY (contrato_id) REFERENCES public.contratos(id);


--
-- Name: organigramas organigramas_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.organigramas
    ADD CONSTRAINT organigramas_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: pap_detalles pap_detalles_version_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pap_detalles
    ADD CONSTRAINT pap_detalles_version_id_fkey FOREIGN KEY (version_id) REFERENCES public.pap_versiones(id) ON DELETE CASCADE;


--
-- Name: planilla_conceptos planilla_conceptos_concepto_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_conceptos
    ADD CONSTRAINT planilla_conceptos_concepto_tenant_id_fkey FOREIGN KEY (concepto_tenant_id) REFERENCES public.conceptos_tenant(id) ON DELETE SET NULL;


--
-- Name: planilla_conceptos planilla_conceptos_planilla_detalle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_conceptos
    ADD CONSTRAINT planilla_conceptos_planilla_detalle_id_fkey FOREIGN KEY (planilla_detalle_id) REFERENCES public.planilla_detalles(id) ON DELETE CASCADE;


--
-- Name: planilla_cts_detalles planilla_cts_detalles_contrato_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_cts_detalles
    ADD CONSTRAINT planilla_cts_detalles_contrato_id_fkey FOREIGN KEY (contrato_id) REFERENCES public.contratos(id) ON DELETE CASCADE;


--
-- Name: planilla_cts_detalles planilla_cts_detalles_planilla_cts_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_cts_detalles
    ADD CONSTRAINT planilla_cts_detalles_planilla_cts_id_fkey FOREIGN KEY (planilla_cts_id) REFERENCES public.planillas_cts(id) ON DELETE CASCADE;


--
-- Name: planilla_detalles planilla_detalles_contrato_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_detalles
    ADD CONSTRAINT planilla_detalles_contrato_id_fkey FOREIGN KEY (contrato_id) REFERENCES public.contratos(id);


--
-- Name: planilla_detalles planilla_detalles_planilla_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planilla_detalles
    ADD CONSTRAINT planilla_detalles_planilla_id_fkey FOREIGN KEY (planilla_id) REFERENCES public.planillas(id) ON DELETE CASCADE;


--
-- Name: planillas planillas_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.planillas
    ADD CONSTRAINT planillas_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: puesto_conceptos puesto_conceptos_concepto_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puesto_conceptos
    ADD CONSTRAINT puesto_conceptos_concepto_tenant_id_fkey FOREIGN KEY (concepto_tenant_id) REFERENCES public.conceptos_tenant(id) ON DELETE CASCADE;


--
-- Name: puesto_conceptos puesto_conceptos_puesto_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puesto_conceptos
    ADD CONSTRAINT puesto_conceptos_puesto_id_fkey FOREIGN KEY (puesto_id) REFERENCES public.puestos(id) ON DELETE CASCADE;


--
-- Name: puestos puestos_fuente_rubro_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos
    ADD CONSTRAINT puestos_fuente_rubro_id_fkey FOREIGN KEY (fuente_rubro_id) REFERENCES public.fuentes_rubros(id);


--
-- Name: puestos puestos_meta_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos
    ADD CONSTRAINT puestos_meta_id_fkey FOREIGN KEY (meta_id) REFERENCES public.metas_presupuestales(id);


--
-- Name: puestos puestos_regimen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos
    ADD CONSTRAINT puestos_regimen_id_fkey FOREIGN KEY (regimen_id) REFERENCES public.regimenes_laborales(id);


--
-- Name: puestos puestos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos
    ADD CONSTRAINT puestos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: puestos puestos_unidad_organica_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.puestos
    ADD CONSTRAINT puestos_unidad_organica_id_fkey FOREIGN KEY (unidad_organica_id) REFERENCES public.unidades_organicas(id) ON DELETE SET NULL;


--
-- Name: regimen_concepto_modelo regimen_concepto_modelo_concepto_modelo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_modelo
    ADD CONSTRAINT regimen_concepto_modelo_concepto_modelo_id_fkey FOREIGN KEY (concepto_modelo_id) REFERENCES public.conceptos_modelo(id) ON DELETE CASCADE;


--
-- Name: regimen_concepto_modelo regimen_concepto_modelo_regimen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_modelo
    ADD CONSTRAINT regimen_concepto_modelo_regimen_id_fkey FOREIGN KEY (regimen_id) REFERENCES public.regimenes_laborales(id) ON DELETE CASCADE;


--
-- Name: regimen_concepto_tenant regimen_concepto_tenant_concepto_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_tenant
    ADD CONSTRAINT regimen_concepto_tenant_concepto_tenant_id_fkey FOREIGN KEY (concepto_tenant_id) REFERENCES public.conceptos_tenant(id) ON DELETE CASCADE;


--
-- Name: regimen_concepto_tenant regimen_concepto_tenant_regimen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_tenant
    ADD CONSTRAINT regimen_concepto_tenant_regimen_id_fkey FOREIGN KEY (regimen_id) REFERENCES public.regimenes_laborales(id) ON DELETE CASCADE;


--
-- Name: regimen_concepto_tenant regimen_concepto_tenant_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.regimen_concepto_tenant
    ADD CONSTRAINT regimen_concepto_tenant_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: trabajadores trabajadores_afp_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT trabajadores_afp_id_fkey FOREIGN KEY (afp_id) REFERENCES public.afps(id);


--
-- Name: trabajadores trabajadores_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT trabajadores_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: unidades_organicas unidades_organicas_organigrama_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unidades_organicas
    ADD CONSTRAINT unidades_organicas_organigrama_id_fkey FOREIGN KEY (organigrama_id) REFERENCES public.organigramas(id) ON DELETE CASCADE;


--
-- Name: unidades_organicas unidades_organicas_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unidades_organicas
    ADD CONSTRAINT unidades_organicas_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.unidades_organicas(id) ON DELETE SET NULL;


--
-- Name: unidades_organicas unidades_organicas_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unidades_organicas
    ADD CONSTRAINT unidades_organicas_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: usuarios usuarios_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict DlIhE8ov2sKGqW0fdEPEhxnbXD3P5JnesY0TcXYbn38arWaMjVMaCNtiWJbXNBi

