package services

import (
	"fmt"
	"io"
	"log"
	"mail/config/configSMTP"
	"mail/models"
	"mime/multipart"
	"os"
	"strings"

	"gopkg.in/gomail.v2"
)

type ServicePort interface {
	MailChicCRMServices(mailRequest models.MailRequest, files []*multipart.FileHeader) (string, error)
}

type serviceAdapter struct {
	// add field for dependency injection
}

func NewServiceAdapter() ServicePort {
	return &serviceAdapter{}
}

func SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	// ทำการ copy ไฟล์จาก src ไปยัง out
	if _, err = io.Copy(out, src); err != nil {
		return err
	}

	return nil
}
func (s *serviceAdapter) MailChicCRMServices(mailRequest models.MailRequest, files []*multipart.FileHeader) (string, error) {
	var attachmentURLs []string
	for _, file := range files {
		attachmentURL := "./" + file.Filename
		if err := SaveUploadedFile(file, attachmentURL); err != nil {
			log.Println(err)
			return "", err
		}
		attachmentURLs = append(attachmentURLs, attachmentURL)
	}

	message := gomail.NewMessage()
	message.SetHeader("From", configSMTP.SMTPUsername)
	// message.SetHeader("To", mailRequest.To)
	message.SetHeader("To", mailRequest.To...)
	message.SetHeader("Subject", mailRequest.Subject)
	message.SetHeader("Reply-To", mailRequest.FromEmail)
	// fmt.Println(mailRequest.CC)

	if len(mailRequest.CC) > 0 {
		var ccAddresses []string
		for _, cc := range mailRequest.CC {
			ccAddresses = append(ccAddresses, strings.Split(strings.TrimSpace(cc), ",")...)
		}
		message.SetHeader("Cc", ccAddresses...)
	}

	bodylinkHTML := fmt.Sprintf("<a href=\"%s\">%s</a>", mailRequest.BodyLink, mailRequest.LinkName)
	message.SetBody("text/html", fmt.Sprintf("%s<br>%s<br>%s<br>%s", mailRequest.Body, mailRequest.Body1, mailRequest.Body2, bodylinkHTML))

	// Attach all the files
	for _, attachmentURL := range attachmentURLs {
		message.Attach(attachmentURL)
	}
	defer func() {
		for _, attachmentURL := range attachmentURLs {
			if rmErr := os.Remove(attachmentURL); rmErr != nil {
				fmt.Printf("Error deleting file: %v\n", rmErr)
			}
		}
	}()
	d := gomail.NewDialer(configSMTP.SMTPServer, configSMTP.SMTPPort, configSMTP.SMTPUsername, configSMTP.SMTPPassword)
	if err := d.DialAndSend(message); err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return "", err
	}
	// for _, attachmentURL := range attachmentURLs { // use defer better for always remove the files
	// 	if err := os.Remove(attachmentURL); err != nil {
	// 		fmt.Printf("Error deleting file: %v\n", err)
	// 	}
	// }
	return "Email sent successfully", nil
}
