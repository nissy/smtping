package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Bowery/prompt"
	"github.com/nissy/smtping"
)

const (
	Subject = "SMTPING"
)

var (
	sessioncnt    = flag.Int("s", 1, "specify a number of cocurrent sessions")
	messagecnt    = flag.Int("m", 1, "specify a number of messages to send")
	domain        = flag.String("d", "", "specify a to domain")
	fromAddress   = flag.String("from", "", "specify a envelope-from")
	databytes     = flag.Int("byte", 10, "specify a number of data bytes")
	address       = flag.String("addr", "", "specify a to smtp address <hostname:port>")
	isAuth        = flag.Bool("auth", false, "enter authentication information")
	isDetail      = flag.Bool("detail", false, "detailed output")
	isDisablePipe = flag.Bool("disable-pipe", false, "disable to pipelining")
	isVersion     = flag.Bool("v", false, "show version and exit")
	isHelp        = flag.Bool("h", false, "this help")
	Version       string
)

func main() {
	os.Exit(exitcode(run()))
}

func exitcode(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 1
	}

	return 0
}

func run() error {
	flag.Parse()

	if *isVersion {
		if len(Version) > 0 {
			fmt.Println("v" + Version)
		}

		return nil
	}

	if *isHelp {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] -d to_domain to_user...\n", os.Args[0])
		flag.PrintDefaults()
		return nil
	}

	users := flag.Args()

	if len(users) == 0 {
		return errors.New("to users is not specified.")
	}

	if len(*domain) == 0 {
		return errors.New("to domain is not specified.")
	}

	if len(*fromAddress) == 0 {
		*fromAddress = users[0] + "@" + *domain
	}

	var auth smtp.Auth

	if *isAuth {
		if len(*address) == 0 {
			var err error

			*address, err = prompt.Basic("SMTP Address <hostname:port>: ", true)

			if err != nil {
				return err
			}
		}

		host, _, err := net.SplitHostPort(*address)

		if err != nil {
			return err
		}

		authUser, err := prompt.Basic("UserName: ", true)

		if err != nil {
			return err
		}

		authPass, err := prompt.Password("Password: ")

		if err != nil {
			return err
		}

		auth = smtp.PlainAuth(
			"",
			authUser,
			authPass,
			host,
		)
	} else {
		mx, err := net.LookupMX(*domain)

		if err != nil {
			return err
		}

		*address = mx[0].Host + ":25"
	}

	fmt.Fprintf(
		os.Stdout,
		"SMTPING %s: %v message data bytes\n",
		*domain,
		*databytes,
	)

	var wg sync.WaitGroup
	timePingStart := time.Now()

	for i := 1; i < *sessioncnt+1; i++ {
		wg.Add(1)
		go func(number int) {
			timeSessionStart := time.Now()
			c, err := smtp.NewClient(*domain, *address, auth)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				wg.Done()
				return
			}

			if *isDisablePipe {
				c.DisableExtension("PIPELINING")
			}

			defer func() {
				c.Close()
				wg.Done()

				fmt.Fprintf(
					os.Stdout,
					"session number=%v host=%s pipe=%t tls=%t time=%v ms\n",
					number,
					c.Host,
					c.IsExtension("PIPELINING"),
					c.IsExtension("STARTTLS"),
					smtp.Processing(timeSessionStart, time.Now()),
				)
			}()

			data := randchar(*databytes)

			for i := 0; i < *messagecnt; i++ {
				PrintMailSend(
					number,
					c.MailSend(*fromAddress, users[i%len(users)], Subject, data),
					*isDetail,
				)
			}
		}(i)
	}

	wg.Wait()

	fmt.Fprintf(
		os.Stdout,
		"smtping time=%v ms\n",
		smtp.Processing(timePingStart, time.Now()),
	)

	return nil
}

func randchar(size int) string {
	r := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, size)

	for i := range b {
		b[i] = r[rand.Intn(len(r))]
	}

	return string(b)
}

func PrintMailSend(number int, m *smtp.Mail, detail bool) {
	p := fmt.Sprintf(
		"mail session=%v code=%v time=%v ms\n",
		number,
		m.Code,
		m.Time,
	)

	if detail {
		for _, v := range m.Commands {
			if v.Response == nil {
				break
			}

			p += fmt.Sprintf("--> "+v.Format+"\n", v.Args...)
			p += fmt.Sprintf("<-- %v %s\n", v.Response.Code, v.Response.Message)
		}

		if m.Error == nil {
			p += fmt.Sprintf("--> %s", m.Content())
			p += fmt.Sprintf("<-- %v %s\n", m.Code, m.Message)
		}

		p += "\n"
	}

	fmt.Fprint(os.Stdout, p)
}
