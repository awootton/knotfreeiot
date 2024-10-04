package mail

import (
	"crypto/tls"
	"net/smtp"
	"strings"
)

func makeEmailMessage(from string, to []string, subject string, body string) string {
	// 	From: someone@example.com
	// To: someone_else@example.com
	// Subject: An RFC 822 formatted message

	// This is the plain text body of the message. Note the blank line
	// between the header information and the body of the message.

	result := "From: " + from + "\r\n"
	result += "To: " + strings.Join(to, ",") + "\r\n"
	//result += "Bcc: " + "nobody@gotohere.com" + "\r\n"
	result += "Subject: " + subject + "\r\n"
	result += "\r\n"
	result += body + "\r\n"

	return result
}

// Mail does not depend on iot but iot will use mail.
// not using the more func?
func MySendMail(addr string, a smtp.Auth, from string, to []string, msg []byte,
	more func(c *smtp.Client) error) error {
	// if err := validateLine(from); err != nil {
	// 	return err
	// }
	// for _, recp := range to {
	// 	if err := validateLine(recp); err != nil {
	// 		return err
	// 	}
	// }
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	// if err = c.hello(); err != nil {
	// 	return err
	// }
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: "smtp.gmail.com"} //c.serverName}
		// if testHookStartTLS != nil {
		// 	testHookStartTLS(config)
		// }
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}
	// if a != nil && c.ext != nil {
	// 	if _, ok := c.ext["AUTH"]; !ok {
	// 		return errors.New("smtp: server doesn't support AUTH")
	// 	}
	if err = c.Auth(a); err != nil {
		return err
	}
	// }
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	err = more(c)
	if err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
