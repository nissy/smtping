package smtp

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strings"
	"time"
)

var (
	ConnTimeOutSecond = 3
)

type Client struct {
	Domain     string
	Host       string
	nConn      net.Conn
	tConn      *textproto.Conn
	extensions map[string]string
	auth       []string
}

type dataCloser struct {
	c *Client
	io.WriteCloser
}

type Command struct {
	ExpectCode int
	Format     string
	Args       []interface{}
	Response   *Response
	Error      error
}

type Response struct {
	Code    int
	Message string
}

type Mail struct {
	Date     time.Time
	Stat     string
	From     string
	To       string
	Subject  string
	Body     string
	Size     int
	Time     int64
	Code     int
	Message  string
	Error    error
	Commands []*Command
}

func NewCommand(expectCode int, format string, args ...interface{}) *Command {
	return &Command{
		ExpectCode: expectCode,
		Format:     format,
		Args:       args,
	}
}

func (c *Client) Exec(cmd ...*Command) error {
	id := c.tConn.Next()

	if len(cmd) > 1 {
		c.tConn.Pipeline.StartRequest(id)
	} else {
		c.tConn.StartRequest(id)
	}

	for _, v := range cmd {
		if err := c.tConn.PrintfLine(v.Format, v.Args...); err != nil {
			return err
		}
	}

	if len(cmd) > 1 {
		c.tConn.Pipeline.EndRequest(id)
		c.tConn.Pipeline.StartResponse(id)
		defer c.tConn.Pipeline.EndResponse(id)
	} else {
		c.tConn.EndRequest(id)
		c.tConn.StartResponse(id)
		defer c.tConn.EndResponse(id)
	}

	for _, v := range cmd {
		v.Response = &Response{}
		v.Response.Code, v.Response.Message, v.Error = c.tConn.ReadResponse(v.ExpectCode)
	}

	return cmd[len(cmd)-1].Error
}

func (m *Mail) Content() (c string) {
	c += "From:" + m.From + "\r\n"
	c += "To:" + m.To + "\r\n"
	c += "Subject:" + m.Subject + "\r\n"
	c += m.Body + "\r\n"
	//c += "Message-ID: <>"

	return c
}

func NewClient(domain, addr string, auth Auth) (*Client, error) {
	host, _, err := net.SplitHostPort(addr)

	if err != nil {
		return nil, err
	}

	c := &Client{
		Domain: domain,
		Host:   host,
	}

	if err := c.connect(addr); err != nil {
		return nil, err
	}

	if err := c.ehlo(); err != nil {
		if err := c.Exec(NewCommand(250, "HELO %s", "localhost")); err != nil {
			return nil, err
		}
	} else {
		if c.IsExtension("STARTTLS") {
			if err := c.Exec(NewCommand(220, "STARTTLS")); err != nil {
				return nil, err
			}

			c.nConn = tls.Client(c.nConn, &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         c.Host,
			})
			c.tConn = textproto.NewConn(c.nConn)
		}

		if auth != nil {
			if err = c.Auth(auth); err != nil {
				return nil, err
			}
		}
	}

	return c, nil
}

func (c *Client) connect(addr string) (err error) {
	if c.nConn, err = net.DialTimeout("tcp", addr, time.Duration(ConnTimeOutSecond)*time.Second); err != nil {
		return err
	}

	c.tConn = textproto.NewConn(c.nConn)

	if _, _, err := c.tConn.ReadResponse(220); err != nil {
		c.tConn.Close()
		return err
	}

	return nil
}

func (c *Client) IsExtension(name string) bool {
	_, ok := c.extensions[name]
	return ok
}

func (c *Client) DisableExtension(name string) {
	delete(c.extensions, name)
}

func (c *Client) ehlo() error {
	cmd := NewCommand(250, "EHLO %s", "localhost")

	if err := c.Exec(cmd); err != nil {
		return err
	}

	c.extensions = make(map[string]string)
	exts := strings.Split(cmd.Response.Message, "\n")

	if len(exts) > 1 {
		for _, line := range exts[1:] {
			args := strings.SplitN(line, " ", 2)
			if len(args) > 1 {
				c.extensions[args[0]] = args[1]
			} else {
				c.extensions[args[0]] = ""
			}
		}
	}

	if mechs, ok := c.extensions["AUTH"]; ok {
		c.auth = strings.Split(mechs, " ")
	}

	return nil
}

func (c *Client) Auth(a Auth) error {
	encoding := base64.StdEncoding

	mech, resp, err := a.Start(
		&ServerInfo{
			c.Host,
			c.IsExtension("STARTTLS"),
			c.auth,
		},
	)

	if err != nil {
		c.Close()
		return err
	}

	resp64 := make([]byte, encoding.EncodedLen(len(resp)))
	encoding.Encode(resp64, resp)

	cmd := NewCommand(0, strings.TrimSpace(fmt.Sprintf("AUTH %s %s", mech, resp64)))

	if err := c.Exec(cmd); err != nil {
		return err
	}

	for err == nil {
		var m []byte

		switch cmd.Response.Code {
		case 334:
			m, err = encoding.DecodeString(cmd.Response.Message)
		case 235:
			m = []byte(cmd.Response.Message)
		default:
			err = &textproto.Error{
				Code: cmd.Response.Code,
				Msg:  cmd.Response.Message,
			}
		}

		if err == nil {
			resp, err = a.Next(m, cmd.Response.Code == 334)
		}

		if err != nil {
			if err = c.Exec(NewCommand(501, "*")); err != nil {
				return err
			}

			c.Close()
			break
		}

		if resp == nil {
			break
		}

		resp64 = make([]byte, encoding.EncodedLen(len(resp)))
		encoding.Encode(resp64, resp)

		cmd := NewCommand(0, string(resp64))

		if err = c.Exec(cmd); err != nil {
			return err
		}
	}

	return err
}

func (c *Client) Close() error {
	if err := c.Exec(NewCommand(221, "QUIT")); err != nil {
		return err
	}

	return c.tConn.Close()
}

func (c *Client) MailSend(from, user, subject, body string) *Mail {
	m := &Mail{
		From:    from,
		To:      user + "@" + c.Domain,
		Subject: subject,
		Body:    body,
		Stat:    "sent",
	}

	m.Commands = []*Command{
		{
			ExpectCode: 250,
			Format:     "MAIL FROM:<%s>%s",
			Args: []interface{}{
				m.From,
				" BODY=8BITMIME",
			},
		},
		{
			ExpectCode: 250,
			Format:     "RCPT TO:<%s>",
			Args: []interface{}{
				m.To,
			},
		},
		{
			ExpectCode: 354,
			Format:     "DATA",
		},
	}

	sTime := time.Now()

	defer func() {
		if m.Error != nil {
			m.Stat = "bounce"

			if e, ok := m.Error.(*textproto.Error); ok {
				m.Code = e.Code
				m.Message = e.Msg
			} else {
				m.Message = m.Error.Error()
			}
		}

		m.Date = time.Now()
		m.Time = Processing(sTime, m.Date)
		m.Size = len(m.Body)
	}()

	if c.IsExtension("PIPELINING") {
		if m.Error = c.Exec(m.Commands...); m.Error != nil {
			return m
		}
	} else {
		for _, v := range m.Commands {
			if m.Error = c.Exec(v); m.Error != nil {
				return m
			}
		}
	}

	wc := &dataCloser{
		c,
		c.tConn.DotWriter(),
	}

	buf := bytes.NewBufferString(m.Content())

	if _, m.Error = buf.WriteTo(wc); m.Error != nil {
		return m
	}

	if m.Error = wc.Close(); m.Error != nil {
		return m
	}

	m.Code, m.Message, m.Error = c.tConn.ReadResponse(250)

	return m
}
