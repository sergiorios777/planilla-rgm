package services

import (
	"log"
	"net/smtp"
	"os"
)

// Mailer define la interfaz para el envío de correos electrónicos
type Mailer interface {
	Enviar(para string, asunto string, cuerpo string) error
}

// SMTPMailer implementa la interfaz Mailer utilizando el servidor de correo real mediante net/smtp
type SMTPMailer struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

// Enviar realiza la entrega real del correo usando SMTP
func (m *SMTPMailer) Enviar(para string, asunto string, cuerpo string) error {
	addr := m.Host + ":" + m.Port
	auth := smtp.PlainAuth("", m.User, m.Password, m.Host)
	msg := []byte("To: " + para + "\r\n" +
		"Subject: " + asunto + "\r\n" +
		"\r\n" +
		cuerpo + "\r\n")

	err := smtp.SendMail(addr, auth, m.From, []string{para}, msg)
	if err != nil {
		log.Printf("SMTPMailer: Error al enviar correo a %s: %v", para, err)
		return err
	}
	log.Printf("SMTPMailer: Correo enviado a %s exitosamente.", para)
	return nil
}

// MockMailer implementa la interfaz Mailer imprimiendo el contenido en consola para desarrollo
type MockMailer struct{}

// Enviar simplemente loguea la información en la consola de depuración
func (m *MockMailer) Enviar(para string, asunto string, cuerpo string) error {
	log.Printf("\n---------------- MOCK EMAIL SENT ----------------\n"+
		"Destinatario : %s\n"+
		"Asunto       : %s\n"+
		"Cuerpo       : %s\n"+
		"--------------------------------------------------", para, asunto, cuerpo)
	return nil
}

// NewMailService es la fábrica que decide qué implementación de Mailer retornar según el entorno (.env)
func NewMailService() Mailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		log.Println("💡 SMTP_HOST no configurada. Iniciando MockMailer (los correos se imprimirán en consola).")
		return &MockMailer{}
	}

	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")

	log.Printf("📧 Iniciando SMTPMailer con host: %s y puerto: %s", host, port)
	return &SMTPMailer{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		From:     from,
	}
}
