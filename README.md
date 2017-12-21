smtping
=====================

smtping is a command to investigate the smtp server.

### Example

#### Basic

```
$ smtping -s 3 -m 3 -d example.com user1 user2 user3
SMTPING example.com: 10 message data bytes
mail session=2 code=250 time=527 ms
mail session=1 code=250 time=541 ms
mail session=3 code=250 time=545 ms
mail session=2 code=250 time=457 ms
mail session=1 code=250 time=456 ms
mail session=3 code=250 time=459 ms
mail session=2 code=250 time=441 ms
mail session=3 code=250 time=445 ms
mail session=1 code=250 time=473 ms
session number=2 host=aspmx.l.example.com. pipe=true tls=true time=2115 ms
session number=3 host=aspmx.l.example.com. pipe=true tls=true time=2154 ms
session number=1 host=aspmx.l.example.com. pipe=true tls=true time=2161 ms
smtping time=2161 ms
```

#### Auth

```
$ smtping -auth -d example.com user
SMTP Address <hostname:port>: smtp.example.com:587
UserName: user@example.com
Password:
SMTPING example.com: 10 message data bytes
mail session=1 code=250 time=1181 ms
session number=1 host=smtp.example.com pipe=true tls=true time=2410 ms
smtping time=2410 ms
```

### Synopsis

```
Usage: smtping [options] -d to_domain to_user...
  -addr string
        specify a to smtp address <hostname:port>
  -byte int
        specify a number of data bytes (default 10)
  -d string
        specify a to domain
  -detail
        detailed output
  -disable-pipe
        disable to pipelining
  -auth
        enter authentication information
  -from string
        specify a envelope-from
  -h    this help
  -m int
        specify a number of messages to send (default 1)
  -s int
        specify a number of cocurrent sessions (default 1)
  -v    show version and exit
```