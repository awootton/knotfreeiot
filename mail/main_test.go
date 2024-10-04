package mail

import (
	"fmt"
	"net/smtp"
	"os"
	"testing"
	"time"
)

// Mail does not depend on iot but iot will use mail.

func TestMail1(t *testing.T) {
	_ = t
	// mail.SendMail()

	// we used environment variables to load the
	// email address and the password from the shell
	// you can also directly assign the email address
	// and the password
	// from := os.Getenv("MAIL")
	// password := os.Getenv("PASSWD")

	password := "" // "password" same as your web password. sorta
	// username := "alan.wootton@gmail.com"

	homeDir, err := os.UserHomeDir()
	_ = err
	fmt.Println("homeDir", homeDir)
	tmp, err := os.ReadFile(homeDir + "/atw/googleapppass.txt")
	_ = err
	password = string(tmp)

	// host is address of server that the
	// sender's email address belongs,
	// in this case its gmail.
	// For e.g if your are using yahoo
	// mail change the address as smtp.mail.yahoo.com
	host := "smtp.gmail.com" //"smtp.mail.yahoo.com" // "smtp.gmail.com"

	// PlainAuth uses the given username and password to
	// authenticate to host and act as identity.
	// Usually identity should be the empty string,
	// to act as username.
	auth := smtp.PlainAuth("alan.wootton@gmail.com", "alan.wootton@gmail.com", password, host)

	from := "alan@gotohere.com"
	// toList is list of email address that email is to be sent.
	toList := []string{"alan@gotohere.com", "mercury@gotohere.com", "alex_woods66@yahoo.com"}

	// Its the default port of smtp server
	port := "587" // "25" hangs - no connection//

	// This is the message to send in the mail
	msg := "Hello from dreaming of knot free mail " + time.Now().String() + "\r\n"

	fmt.Println("from", from)
	fmt.Println("to", toList)
	fmt.Println("msg", msg)

	// We can't send strings directly in mail,
	// strings need to be converted into slice bytes
	// body := []byte(msg)

	// The msg parameter should be an RFC 822-style email with headers
	// first, a blank line, and then the message body. The lines of msg
	// should be CRLF terminated. The msg headers should usually include
	// fields such as "From", "To", "Subject", and "Cc".  Sending "Bcc"
	// messages is accomplished by including an email address in the to
	// parameter but not including it in the msg headers.

	title := "Hello from knot free mail forward" // \r\n

	emailMessage := makeEmailMessage(from, toList, title, msg)

	fmt.Println("emailMessage", emailMessage)

	// SendMail uses TLS connection to send the mail
	// The email is sent to all address in the toList,
	// the body should be of type bytes, not strings
	// This returns error if any occurred.
	err = smtp.SendMail(host+":"+port, auth, from, toList, []byte(emailMessage))
	// err = MySendMail(host+":"+port, auth, from, toList, []byte(emailMessage), func(c *smtp.Client) error {
	// 	fmt.Println("in MySendMail") // we're not using this
	// 	return nil
	// })

	// handling the errors
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Successfully sent mail to all user in toList")
}
